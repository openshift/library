package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"regexp"
	"strings"
	"sync"

	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"

	"k8s.io/klog"

	imageapiv1 "github.com/openshift/api/image/v1"
	templateapiv1 "github.com/openshift/api/template/v1"
	libraryapiv1 "github.com/openshift/library/api/library/v1"
)

var (
	dir      string
	tags     []string
	matchAll bool
	loglevel int
	sources  sync.Map
)

// importCmd represents the import command
var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Imports the imagestreams and templates defined in the YAML library documents.",
	Long: `
	Examples:
	// import everything to the default location
	$ library import

	// import the data into a specific directory
	$ library import --dir somedir

	// import only the specified document(s)
	$ library import --documents document.yaml,otherdocument.yaml

	// import only imagestreams and templates that match one or more of the specified tags
	$ library import --tags tag1,tag2,tag3

	// import only the imagestreams and templates that match ALL of the specified tags
	$ library import --tags tag1,tag2,tag3 --match-all-tags
	`,
	Run: func(cmd *cobra.Command, args []string) {
		errorChan := make(chan error, 1000)
		if len(dir) != 0 {
			if _, err := os.Stat(dir); os.IsNotExist(err) {
				os.MkdirAll(dir, os.ModePerm)
			}
		}
		var wg sync.WaitGroup
		documents, err := rootCmd.Flags().GetStringSlice("documents")
		if err != nil {
			klog.Errorf("Unable to get documents: %v", err)
			os.Exit(1)
		}
		for _, document := range documents {
			if strings.HasSuffix(document, ".yaml") {
				document = strings.Replace(document, ".yaml", "", 1)
			}

			// Delete the old directory
			os.RemoveAll(path.Join(dir, document))

			// Read the YAML file
			contents, err := ioutil.ReadFile(fmt.Sprintf("%s.yaml", document))
			if err != nil {
				klog.Errorf("Error reading file %s.yaml: %v", document, err)
				os.Exit(1)
			}

			// Unmarshal just the variables from the contents
			var variables libraryapiv1.Document
			if err = yaml.Unmarshal([]byte(contents), &variables); err != nil {
				klog.Errorf("Error unmarshalling yaml in file %q replacement: %s", document, err)
				os.Exit(1)
			}

			// Replace variable keys with their values in the contents
			if err := replaceVariables(&contents, variables.Variables); err != nil {
				klog.Errorf("Unable to replace variables in %q, %v", document, err)
				os.Exit(1)
			}

			// Unmarshal the data from the contents
			documentData := libraryapiv1.DocumentData{}
			if err = yaml.Unmarshal([]byte(contents), &documentData); err != nil {
				klog.Errorf("Error unmarshalling yaml after variable replacement: %s", err)
				os.Exit(1)
			}
			for folder, item := range documentData.Data {
				// If this item has imagestreams, create the directory
				if len(item.ImageStreams) != 0 {
					isPath := path.Join(dir, document, folder, "imagestreams")
					if _, err := os.Stat(isPath); os.IsNotExist(err) {
						if err := os.MkdirAll(isPath, os.ModePerm); err != nil {
							klog.Errorf("Error creating directory %q: %v", isPath, err)
							os.Exit(1)
						}
					}
					wg.Add(1)
					go processImagestreams(&wg, document, folder, isPath, item.ImageStreams, errorChan)
				}
				// If this item has templates, create the directory
				if len(item.Templates) != 0 {
					tPath := path.Join(dir, document, folder, "templates")
					if _, err := os.Stat(tPath); os.IsNotExist(err) {
						if err := os.MkdirAll(tPath, os.ModePerm); err != nil {
							klog.Errorf("Error creating directory %q: %v", tPath, err)
							os.Exit(1)
						}
					}
					wg.Add(1)
					go processTemplates(&wg, document, folder, tPath, item.Templates, errorChan)
				}
			}
		}
		wg.Wait()
		close(errorChan)
		for e := range errorChan {
			klog.Errorf("%s", e)
		}

	},
}

func init() {
	rootCmd.AddCommand(importCmd)

	importCmd.Flags().StringSliceVar(&tags, "tags", []string{}, "Select only content with at least one of the specified tag(s) to import templates/imagestreams (separated by comma ',')")
	importCmd.Flags().BoolVar(&matchAll, "match-all-tags", false, "Select only content with all specified tags to import templates/imagestreams (separated by comma ',')")
	importCmd.Flags().StringVar(&dir, "dir", "", "Specify a target directory for the imported content")
	importCmd.Flags().IntVar(&loglevel, "loglevel", 0, "Specify the loglevel.")
}

func hasTag(itemTags []string, filterTags []string, matchAll bool) bool {
	if len(filterTags) == 0 {
		return true
	}

	archFound := false

	for _, tag := range itemTags {
		if strings.HasPrefix(tag, "arch_") {
			archFound = true
		}
	}

	if !archFound {
		itemTags = append(itemTags, "arch_x86_64")
	}

	if !matchAll {
		for _, itemTag := range itemTags {
			for _, filterTag := range filterTags {
				if itemTag == filterTag {
					return true
				}
			}
		}
		return false
	}

	if len(itemTags) != 0 {
		return reflect.DeepEqual(itemTags, filterTags)
	}

	return false
}

func processImagestreams(wg *sync.WaitGroup, document string, folder string, isPath string, imagestreams []libraryapiv1.ItemImageStream, errorChan chan<- error) {
	defer wg.Done()
	for _, imagestream := range imagestreams {
		if !hasTag(imagestream.Tags, tags, matchAll) {
			continue
		}
		foundImageStreams := make([]imageapiv1.ImageStream, 1)
		body, err := fetchURL(imagestream.Location)
		if err != nil {
			errorChan <- fmt.Errorf("Error fetching %s imagestream url %q: %v", document, imagestream.Location, err)
			continue
		}
		is := imageapiv1.ImageStream{}
		if err := unMarshalImageStream(body, &is); err != nil {
			errorChan <- err
		}
		if is.Kind == "ImageStream" {
			foundImageStreams = append(foundImageStreams, is)
		} else if is.Kind == "List" || is.Kind == "ImageStreamList" {
			isl := imageapiv1.ImageStreamList{}
			if err := unMarshalImageStreamList(body, &isl); err != nil {
				errorChan <- err
			}
			for _, item := range isl.Items {
				foundImageStreams = append(foundImageStreams, item)
			}
		}
		for _, stream := range foundImageStreams {
			var match bool
			if len(imagestream.Regex) != 0 {
				match, _ = regexp.MatchString(imagestream.Regex, stream.Name)
			}
			if len(stream.Name) != 0 && (match || len(imagestream.Regex) == 0) {
				fileName := stream.Name
				if len(imagestream.Suffix) != 0 {
					fileName = fmt.Sprintf("%s-%s", stream.Name, imagestream.Suffix)
				}
				imageStreamPath := path.Join(isPath, fmt.Sprintf("%s.json", fileName))
				data, err := json.MarshalIndent(stream, "", "\t")
				if err != nil {
					errorChan <- fmt.Errorf("Error marshaling imagestream %q to json: %v", stream.Name, err)
				}
				if err := writeToFile(data, imageStreamPath); err != nil {
					errorChan <- err
				}
			}
		}
	}
}

func processTemplates(wg *sync.WaitGroup, document string, folder string, tPath string, templates []libraryapiv1.ItemTemplate, errorChan chan<- error) {
	defer wg.Done()
	for _, template := range templates {
		if !hasTag(template.Tags, tags, matchAll) {
			continue
		}

		body, err := fetchURL(template.Location)
		if err != nil {
			errorChan <- fmt.Errorf("Error fetching %s template url %q: %v", document, template.Location, err)
			continue
		}
		t := templateapiv1.Template{}
		if err := unMarshalTemplate(body, &t); err != nil {
			errorChan <- err
		}
		var match bool
		if len(template.Regex) != 0 {
			match, _ = regexp.MatchString(template.Regex, t.Name)
		}
		if len(t.Name) == 0 {
			errorChan <- fmt.Errorf("Template location: %#v", template.Location)
		}
		if len(t.Name) != 0 && (match || len(template.Regex) == 0) {
			fileName := t.Name
			if len(template.Suffix) != 0 {
				fileName = fmt.Sprintf("%s-%s", t.Name, template.Suffix)
			}
			templatePath := path.Join(tPath, fmt.Sprintf("%s.json", fileName))
			data, err := json.MarshalIndent(t, "", "\t")
			if err != nil {
				errorChan <- fmt.Errorf("Error marshaling template %q to json: %v", t.Name, err)
			}
			if err := writeToFile(data, templatePath); err != nil {
				errorChan <- err
			}
		}
	}
}

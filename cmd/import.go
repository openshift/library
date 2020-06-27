package cmd

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"k8s.io/klog"

	imageapiv1 "github.com/openshift/api/image/v1"
	templateapiv1 "github.com/openshift/api/template/v1"
	libraryapiv1 "github.com/openshift/library/api/library/v1"
)

var (
	dir      string
	tags     []string
	matchAll bool
)

// importCmd represents the import command
var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Imports the imagestreams and templates defined in the specified YAML documents.",
	Long: `
	Examples:
	// show the help, same as library import -h
	$ library import
	
	// increase the log level 0, 2, 5, 8
	$ library -v <level>

	// import the data into a specific directory
	$ library import --dir somedir

	// import only the specified document(s)
	// must be a comma separated list
	$ library import --documents document.yaml,otherdocument.yaml

	// import only imagestreams and templates that match ANY of the specified tags
	$ library import --tags tag1,tag2,tag3

	// import only imagestreams and templates that match ALL of the specified tags
	$ library import --tags tag1,tag2,tag3 --match-all-tags
	`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(dir) != 0 {
			if _, err := os.Stat(dir); os.IsNotExist(err) {
				if err := os.MkdirAll(dir, os.ModePerm); err != nil {
					klog.Errorf("unable to create output directory %q: %v", dir, err)
					os.Exit(1)
				}
			}
		}
		var wg sync.WaitGroup
		documents, err := rootCmd.Flags().GetStringSlice("documents")
		if err != nil {
			klog.Errorf("unable to get documents: %v", err)
			os.Exit(1)
		}
		for _, document := range documents {
			klog.Infof("[%s] Processing ...", document)
			if strings.HasSuffix(document, ".yaml") {
				document = strings.Replace(document, ".yaml", "", 1)
			}

			// Delete the old directory
			if err := os.RemoveAll(path.Join(dir, document)); err != nil {
				klog.Errorf("unable to remove directory %q: %v", path.Join(dir, document), err)
				os.Exit(1)
			}

			// Read the YAML file
			contents, err := ioutil.ReadFile(fmt.Sprintf("%s.yaml", document))
			if err != nil {
				klog.Errorf("[%s] unable to read specified file: %v", document, err)
				os.Exit(1)
			}
			// Convert the YAML to JSON
			contents, err = yaml.YAMLToJSON(contents)
			klog.V(5).Infof("[%s] Converting yaml to json", document)
			if err != nil {
				klog.Errorf("[%s] unable to convert yaml to json: %v", document, err)
				os.Exit(1)
			}
			// Unmarshal just the variables from the contents
			var variables libraryapiv1.Document
			if err = yaml.Unmarshal(contents, &variables); err != nil {
				klog.Errorf("[%s] unable to unMarshal variable replacements: %s", document, err)
				os.Exit(1)
			}

			// Replace variable keys with their values in the contents
			klog.Infof("[%s] Processing variable replacements ...", document)
			if err := replaceVariables(document, &contents, variables.Variables); err != nil {
				klog.Errorf("[%s] unable to replace variables: %v", document, err)
				os.Exit(1)
			}

			// Unmarshal the data from the contents
			documentData := libraryapiv1.DocumentData{}
			if err = yaml.Unmarshal(contents, &documentData); err != nil {
				klog.Errorf("[%s] unable to unMarshal yaml after variable replacement: %v", document, err)
				os.Exit(1)
			}
			for folder, item := range documentData.Data {
				klog.Infof("[%s] Processing folder %q", document, folder)
				if len(item.ImageStreams) != 0 {
					isPath := path.Join(dir, document, folder, "imagestreams")
					wg.Add(1)
					go processImagestreams(&wg, document, folder, isPath, item.ImageStreams)
				}
				if len(item.Templates) != 0 {
					tPath := path.Join(dir, document, folder, "templates")
					wg.Add(1)
					go processTemplates(&wg, document, folder, tPath, item.Templates)
				}
			}
		}
		wg.Wait()
	},
}

func init() {
	rootCmd.AddCommand(importCmd)

	klog.InitFlags(nil)
	flag.Parse()
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)

	importCmd.Flags().StringSliceVar(&tags, "tags", []string{}, "Select only content with at least one of the specified tag(s) to import templates/imagestreams (separated by comma ',')")
	importCmd.Flags().BoolVar(&matchAll, "match-all-tags", false, "Select only content with all specified tags to import templates/imagestreams (separated by comma ',')")
	importCmd.Flags().StringVar(&dir, "dir", "", "Specify a target directory for the imported content")
}

func hasTag(document string, location string, itemTags []string, filterTags []string, matchAll bool) bool {
	if len(filterTags) == 0 {
		return true
	}

	klog.V(8).Infof("[%s] Checking Tags - MatchAll: %t, FilterTags: %#v, ItemTags: %#v, Location: %s", document, matchAll, filterTags, itemTags, location)
	if len(filterTags) != 0 && len(itemTags) == 0 {
		return false
	}

	archFound := false

	for _, tag := range itemTags {
		if strings.HasPrefix(tag, "arch_") {
			archFound = true
		}
	}

	// If no architecture tag is present, set a default of arch_x86_64
	if !archFound {
		klog.V(8).Infof("[%s] No arch tag found, appending %q for %s", document, "arch_x86_64", location)
		itemTags = append(itemTags, "arch_x86_64")
	}

	// If we sort the slices, and there is going to be a match
	// chances are we will find it faster
	// We also need filterTags sorted for comparison with foundTags
	// if we need to match all of the tags
	sort.Strings(itemTags)
	sort.Strings(filterTags)

	var foundTags []string

	for _, filterTag := range filterTags {
		for _, itemTag := range itemTags {
			if itemTag == filterTag {
				// If we don't have to match all tags, and we find a match, return true
				if !matchAll {
					klog.V(8).Infof("[%s] Found matching tag %q for %s", document, filterTag, location)
					return true
				}
				// If we have to match all tags, append the found tag to foundTags
				foundTags = append(foundTags, filterTag)
			}
		}
	}

	// Sort foundTags so we can compare it with filterTags
	sort.Strings(foundTags)

	matchedAllTags := reflect.DeepEqual(foundTags, filterTags)
	if matchedAllTags {
		klog.V(8).Infof("[%s] MatchAllTags: %t, Included %q, ItemTags: %#v FilterTags: %#v", document, matchAll, location, itemTags, filterTags)
	} else {
		klog.V(8).Infof("[%s] MatchAllTags: %t, Skipped %q, ItemTags: %#v FilterTags: %#v", document, matchAll, location, itemTags, filterTags)
	}
	return matchedAllTags
}

func processImagestreams(wg *sync.WaitGroup, document string, folder string, isPath string, imagestreams []libraryapiv1.ItemImageStream) {
	defer wg.Done()
	for _, imagestream := range imagestreams {
		if !hasTag(document, imagestream.Location, imagestream.Tags, tags, matchAll) {
			klog.V(5).Infof("[%s] Tags do not match, skipping %s", document, imagestream.Location)
			continue
		}
		foundImageStreams := make([]imageapiv1.ImageStream, 1)
		body, err := fetchURL(document, imagestream.Location)
		if err != nil {
			klog.Errorf("[%s] unable to fetch imagestream url %q: %v", document, imagestream.Location, err)
			continue
		}
		is := imageapiv1.ImageStream{}
		if err := unMarshalImageStream(body, &is); err != nil {
			klog.Errorf("[%s] unable to unMarshal imagestream %q: %v", imagestream.Location, err)
		}
		if is.Kind == "ImageStream" {
			foundImageStreams = append(foundImageStreams, is)
		} else if is.Kind == "List" || is.Kind == "ImageStreamList" {
			isl := imageapiv1.ImageStreamList{}
			if err := unMarshalImageStreamList(body, &isl); err != nil {
				klog.Errorf("[%s] unable to unMarshal imagestream list: %v:", document, err)
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
				klog.Infof("[%s] Processing imagestream %q", document, stream.Name)
				fileName := stream.Name
				if len(imagestream.Suffix) != 0 {
					fileName = fmt.Sprintf("%s-%s", stream.Name, imagestream.Suffix)
				}
				imageStreamPath := path.Join(isPath, fmt.Sprintf("%s.json", fileName))
				data, err := json.MarshalIndent(stream, "", "\t")
				if err != nil {
					klog.Errorf("[%s] unable to marshal imagestream %q to json: %v", document, stream.Name, err)
				}
				if err := writeToFile(document, data, imageStreamPath); err != nil {
					klog.Errorf("[%s] unable to write data for %q to file: %v", document, stream.Name, err)
				}
			}
		}
	}
}

func processTemplates(wg *sync.WaitGroup, document string, folder string, tPath string, templates []libraryapiv1.ItemTemplate) {
	defer wg.Done()
	for _, template := range templates {
		if !hasTag(document, template.Location, template.Tags, tags, matchAll) {
			klog.V(5).Infof("[%s] Tags do not match, skipping %s", document, template.Location)
			continue
		}

		body, err := fetchURL(document, template.Location)
		if err != nil {
			klog.Errorf("[%s] unable to fetch template url %q: %v", document, template.Location, err)
			continue
		}
		t := templateapiv1.Template{}
		if err := unMarshalTemplate(body, &t); err != nil {
			klog.Errorf("[%s] unable to unMarshal template %q: %v", template.Location, err)
		}
		var match bool
		if len(template.Regex) != 0 {
			match, _ = regexp.MatchString(template.Regex, t.Name)
		}
		if len(t.Name) != 0 && (match || len(template.Regex) == 0) {
			klog.Infof("[%s] Processing template %q", document, t.Name)
			fileName := t.Name
			if len(template.Suffix) != 0 {
				fileName = fmt.Sprintf("%s-%s", t.Name, template.Suffix)
			}
			templatePath := path.Join(tPath, fmt.Sprintf("%s.json", fileName))
			data, err := json.MarshalIndent(t, "", "\t")
			if err != nil {
				klog.Errorf("[%s] unable to marshal template %q to json: %v", document, t.Name, err)
			}
			if err := writeToFile(document, data, templatePath); err != nil {
				klog.Errorf("[%s] unable to write data for %q to file: %v", document, t.Name, err)
			}
		}
	}
}

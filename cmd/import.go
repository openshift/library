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

	"k8s.io/klog/v2"

	imageapiv1 "github.com/openshift/api/image/v1"
	templateapiv1 "github.com/openshift/api/template/v1"

	libraryapiv1 "github.com/openshift/library/api/library/v1"
)

var (
	config        string
	documents     []string
	dir           string
	tags          []string
	matchAll      bool
	urlCache      sync.Map
	documentCache sync.Map
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

	// specify an import configuration file to use
	$ library import --config configs/config.yaml
	`,
	Run: func(cmd *cobra.Command, args []string) {
		var configs libraryapiv1.Configs
		var wg sync.WaitGroup

		// If we have a configuration, process it
		if len(config) != 0 {
			if _, err := os.Stat(config); os.IsNotExist(err) {
				klog.Errorf("the configuration file specified does not exist: %v", err)
				os.Exit(1)
			}
			c, err := ioutil.ReadFile(config)
			if err != nil {
				klog.Errorf("[%s] unable to read configuration file: %v", config, err)
				os.Exit(1)
			}
			// Convert the YAML to JSON
			c, err = yaml.YAMLToJSON(c)

			if err = json.Unmarshal(c, &configs); err != nil {
				klog.Errorf("[%s] unable to unMarshal configuration: %s", c, err)
				os.Exit(1)
			}
		} else if len(documents) != 0 {
			// If documents are passed, create a configuration from them
			config := libraryapiv1.Config{
				Documents:    documents,
				Tags:         tags,
				MatchAllTags: matchAll,
				OutputDir:    dir,
			}
			configs.Configs = append(configs.Configs, config)
		} else {
			// If no configuration or documents are passed, exit.
			klog.Errorf("Please specify a configuration file or documents to process")
			os.Exit(1)
		}

		if err := processDocuments(&documentCache, configs); err != nil {
			klog.Errorf("unable to process documents: %v", err)
			os.Exit(1)
		}
		if err := preloadCache(&urlCache, &documentCache); err != nil {
			klog.Errorf("unable to preload the cache: %v", err)
			os.Exit(1)
		}
		// Process each configuration
		for i, config := range configs.Configs {
			wg.Add(1)
			sort.Strings(config.Tags)
			go func(i int, config libraryapiv1.Config) {
				defer wg.Done()
				klog.Infof("[Config %d] Processing ...", i)
				// Process each document specified in the configuration
				for _, document := range config.Documents {
					if strings.HasSuffix(document, ".yaml") {
						document = strings.Replace(document, ".yaml", "", 1)
					}
					klog.Infof("[Config %d] Checking directory %q", i, path.Join(config.OutputDir, document))
					_, err := os.Stat(path.Join(config.OutputDir, document))
					if !os.IsNotExist(err) {
						klog.Infof("[Config %d] Removing directory %q", i, path.Join(config.OutputDir, document))
						if err := os.RemoveAll(path.Join(config.OutputDir, document)); err != nil {
							klog.Errorf("[Config %d] unable to remove directory %q: %v", i, path.Join(config.OutputDir, document), err)
							os.Exit(1)
						}
					}
					wg.Add(1)
					go func(document string) {
						defer wg.Done()
						if strings.HasSuffix(document, ".yaml") {
							document = strings.Replace(document, ".yaml", "", 1)
						}
						contents, ok := documentCache.Load(document)
						if !ok {
							klog.Errorf("unable to find document %q in cache", document)
							os.Exit(1)
						}
						// Unmarshal the data from the contents
						documentData := libraryapiv1.DocumentData{}
						if err := json.Unmarshal(contents.([]byte), &documentData); err != nil {
							klog.Errorf("[Config: %d, Doc: %s] unable to unMarshal after variable replacement: %v", i, document, err)
							os.Exit(1)
						}
						for folder, item := range documentData.Data {
							klog.Infof("[Config: %d, Doc: %s] Processing folder %q", i, document, folder)
							if len(item.ImageStreams) != 0 {
								isPath := path.Join(config.OutputDir, document, folder, "imagestreams")
								wg.Add(1)
								go processImagestreams(&wg, i, document, folder, isPath, config.Tags, config.MatchAllTags, item.ImageStreams)
							}
							if len(item.Templates) != 0 {
								tPath := path.Join(config.OutputDir, document, folder, "templates")
								wg.Add(1)
								go processTemplates(&wg, i, document, folder, tPath, config.Tags, config.MatchAllTags, item.Templates)
							}
						}
					}(document)
				}

			}(i, config)
		}

		wg.Wait()
	},
}

func init() {
	rootCmd.AddCommand(importCmd)

	klog.InitFlags(nil)
	flag.Parse()
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)

	importCmd.Flags().StringVar(&config, "config", "", "A JSON or YAML configuration file")
	importCmd.Flags().StringSliceVar(&documents, "documents", []string{}, "The documents to process (comma separated ',')")
	importCmd.Flags().StringSliceVar(&tags, "tags", []string{}, "Select only content with at least one of the specified tag(s) to import templates/imagestreams (separated by comma ',')")
	importCmd.Flags().BoolVar(&matchAll, "match-all-tags", false, "Select only content with all specified tags to import templates/imagestreams (separated by comma ',')")
	importCmd.Flags().StringVar(&dir, "dir", "", "Specify a target directory for the imported content")
}

func hasTag(config int, document string, location string, itemTags []string, filterTags []string, matchAll bool) bool {
	if len(filterTags) == 0 {
		return true
	}

	klog.V(8).Infof("[Config: %d, Doc: %s] Checking Tags - MatchAll: %t, FilterTags: %#v, ItemTags: %#v, Location: %s", config, document, matchAll, filterTags, itemTags, location)
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
		klog.V(8).Infof("[Config: %d, Doc: %s] No arch tag found, appending %q for %s", config, document, "arch_x86_64", location)
		itemTags = append(itemTags, "arch_x86_64")
	}

	// If we sort the slices, and there is going to be a match
	// chances are we will find it faster
	// We also need filterTags sorted for comparison with foundTags
	// if we need to match all of the tags
	sort.Strings(itemTags)

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

func processImagestreams(wg *sync.WaitGroup, config int, document string, folder string, isPath string, tags []string, matchAll bool, imagestreams []libraryapiv1.ItemImageStream) {
	defer wg.Done()
	for _, imagestream := range imagestreams {
		wg.Add(1)
		go func(wg *sync.WaitGroup, document string, folder string, isPath string, imagestream libraryapiv1.ItemImageStream) {
			defer wg.Done()
			if !hasTag(config, document, imagestream.Location, imagestream.Tags, tags, matchAll) {
				klog.V(5).Infof("Config: %d, Doc: %s] Tags do not match, skipping %s", config, document, imagestream.Location)
				return
			}
			foundImageStreams := make([]imageapiv1.ImageStream, 1)
			body, err := fetchURL(&urlCache, imagestream.Location)
			if err != nil {
				klog.Errorf("[Config: %d, Doc: %s] unable to fetch imagestream url %q: %v", config, document, imagestream.Location, err)
				return
			}
			is := imageapiv1.ImageStream{}
			if err := unMarshalImageStream(body, &is); err != nil {
				klog.Errorf("[Config: %d, Doc: %s] unable to unMarshal imagestream %q: %v", config, document, imagestream.Location, err)
			}
			if is.Kind == "ImageStream" {
				foundImageStreams = append(foundImageStreams, is)
			} else if is.Kind == "List" || is.Kind == "ImageStreamList" {
				isl := imageapiv1.ImageStreamList{}
				if err := unMarshalImageStreamList(body, &isl); err != nil {
					klog.Errorf("[Config: %d, Doc: %s] unable to unMarshal imagestream list: %v:", config, document, err)
				}
				for _, item := range isl.Items {
					foundImageStreams = append(foundImageStreams, item)
				}
			}
			for _, stream := range foundImageStreams {
				wg.Add(1)
				go func(wg *sync.WaitGroup, document string, folder string, isPath string, imagestream libraryapiv1.ItemImageStream, stream imageapiv1.ImageStream) {
					defer wg.Done()
					var match bool
					if len(imagestream.Regex) != 0 {
						match, _ = regexp.MatchString(imagestream.Regex, stream.Name)
					}
					if len(stream.Name) != 0 && (match || len(imagestream.Regex) == 0) {
						klog.Infof("[Config: %d, Doc: %s] Processing imagestream %q", config, document, stream.Name)
						fileName := stream.Name
						if len(imagestream.Suffix) != 0 {
							fileName = fmt.Sprintf("%s-%s", stream.Name, imagestream.Suffix)
						}
						imageStreamPath := path.Join(isPath, fmt.Sprintf("%s.json", fileName))
						data, err := json.MarshalIndent(stream, "", "\t")
						if err != nil {
							klog.Errorf("[Config: %d, Doc: %s] unable to marshal imagestream %q to json: %v", config, document, stream.Name, err)
						}
						if err := writeToFile(config, document, data, imageStreamPath); err != nil {
							klog.Errorf("[Config: %d, Doc: %s] unable to write data for %q to file: %v", config, document, stream.Name, err)
						}
					}
				}(wg, document, folder, isPath, imagestream, stream)
			}
		}(wg, document, folder, isPath, imagestream)
	}
}

func processTemplates(wg *sync.WaitGroup, config int, document string, folder string, tPath string, tags []string, matchAll bool, templates []libraryapiv1.ItemTemplate) {
	defer wg.Done()
	for _, template := range templates {
		wg.Add(1)
		go func(wg *sync.WaitGroup, document string, folder string, isPath string, template libraryapiv1.ItemTemplate) {
			defer wg.Done()
			if !hasTag(config, document, template.Location, template.Tags, tags, matchAll) {
				klog.V(5).Infof("[Config: %d, Doc: %s] Tags do not match, skipping %s", config, document, template.Location)
				return
			}
			body, err := fetchURL(&urlCache, template.Location)
			if err != nil {
				klog.Errorf("[Config: %d, Doc: %s] unable to fetch template url %q: %v", config, document, template.Location, err)
				return
			}
			t := templateapiv1.Template{}
			if err := unMarshalTemplate(body, &t); err != nil {
				klog.Errorf("[Config: %d, Doc: %s] unable to unMarshal template %q: %v", config, document, template.Location, err)
			}
			var match bool
			if len(template.Regex) != 0 {
				match, _ = regexp.MatchString(template.Regex, t.Name)
			}
			if len(t.Name) != 0 && (match || len(template.Regex) == 0) {
				klog.Infof("[Config: %d, Doc: %s] Processing template %q", config, document, t.Name)
				fileName := t.Name
				if len(template.Suffix) != 0 {
					fileName = fmt.Sprintf("%s-%s", t.Name, template.Suffix)
				}
				templatePath := path.Join(tPath, fmt.Sprintf("%s.json", fileName))
				data, err := json.MarshalIndent(t, "", "\t")
				if err != nil {
					klog.Errorf("[Config: %d, Doc: %s] unable to marshal template %q to json: %v", config, document, t.Name, err)
				}
				if err := writeToFile(config, document, data, templatePath); err != nil {
					klog.Errorf("[Config: %d, Doc: %s] unable to write data for %q to file: %v", config, document, t.Name, err)
				}
			}
		}(wg, document, folder, tPath, template)
	}
}

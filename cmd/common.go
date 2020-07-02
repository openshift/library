package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/ghodss/yaml"

	"k8s.io/klog"

	imageapiv1 "github.com/openshift/api/image/v1"
	templateapiv1 "github.com/openshift/api/template/v1"

	libraryapiv1 "github.com/openshift/library/api/library/v1"
)

func processDocuments(cache *sync.Map, configs libraryapiv1.Configs) error {
	var wg sync.WaitGroup
	documents := map[string]struct{}{}
	for _, config := range configs.Configs {

		for _, document := range config.Documents {
			if strings.HasSuffix(document, ".yaml") {
				document = strings.Replace(document, ".yaml", "", 1)
			}
			documents[document] = struct{}{}
		}
	}
	for document := range documents {
		klog.Infof("%v", document)
		wg.Add(1)
		go func(document string) {
			defer wg.Done()
			klog.Infof("[Doc: %s] Processing ...", document)
			if strings.HasSuffix(document, ".yaml") {
				document = strings.Replace(document, ".yaml", "", 1)
			}

			// Read the YAML file
			contents, err := ioutil.ReadFile(fmt.Sprintf("%s.yaml", document))
			if err != nil {
				klog.Errorf("[Doc: %s] unable to read specified file: %v", document, err)
				os.Exit(1)
			}
			// Convert the YAML to JSON
			contents, err = yaml.YAMLToJSON(contents)
			klog.V(5).Infof("[Doc: %s] Converting yaml to json", document)
			if err != nil {
				klog.Errorf("[Doc: %s] unable to convert yaml to json: %v", document, err)
				os.Exit(1)
			}
			// Unmarshal just the variables from the contents
			var variables libraryapiv1.Document
			if err = json.Unmarshal(contents, &variables); err != nil {
				klog.Errorf("[Doc: %s] unable to unMarshal variable replacements: %s", document, err)
				os.Exit(1)
			}

			// Replace variable keys with their values in the contents
			klog.Infof("[Doc: %s] Processing variable replacements ...", document)
			if err := replaceVariables(document, &contents, variables.Variables); err != nil {
				klog.Errorf("[Doc: %s] unable to replace variables: %v", document, err)
				os.Exit(1)
			}

			cache.Store(document, contents)
		}(document)
	}
	wg.Wait()
	return nil

}

func preloadCache(urlCache *sync.Map, documentCache *sync.Map) error {
	var wg sync.WaitGroup
	documentCache.Range(func(k interface{}, v interface{}) bool {
		wg.Add(1)
		go func(k interface{}, v interface{}, urlCache *sync.Map) {
			defer wg.Done()
			documentData := libraryapiv1.DocumentData{}
			if err := json.Unmarshal(v.([]byte), &documentData); err != nil {
				klog.Errorf("[Doc: %s] unable to unMarshal after variable replacement: %v", k.(string), err)
				os.Exit(1)
			}
			for folder, item := range documentData.Data {
				klog.Infof("[Doc: %s] Processing folder %q", k.(string), folder)
				if len(item.ImageStreams) != 0 {
					for _, is := range item.ImageStreams {
						wg.Add(1)
						go func(location string) {
							defer wg.Done()
							if _, err := fetchURL(urlCache, location); err != nil {
								klog.Errorf("unable to fetch url during preload %s: %v", is.Location, err)
							}
						}(is.Location)
					}
				}
				if len(item.Templates) != 0 {
					for _, t := range item.Templates {
						wg.Add(1)
						go func(location string) {
							defer wg.Done()
							if _, err := fetchURL(urlCache, location); err != nil {
								klog.Errorf("unable to fetch url during preload %s: %v", t.Location, err)
							}
						}(t.Location)
					}
				}
			}

		}(k, v, urlCache)

		return true
	})
	wg.Wait()

	return nil
}

func writeToFile(config int, document string, data []byte, filePath string) error {
	klog.V(5).Infof("[Config: %d, Doc: %s] Writing file %s", config, document, filePath)
	if _, err := os.Stat(filepath.Dir(filePath)); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
			return fmt.Errorf("Error creating directory %q: %v", filepath.Dir(filePath), err)
		}
	}
	return ioutil.WriteFile(filePath, data, os.ModePerm)
}

func replaceVariables(document string, d *[]byte, v map[string]string) error {
	for k, v := range v {
		klog.V(5).Infof("[Doc: %s] Replacing variable {%s} with %s", document, k, v)
		*d = []byte(strings.ReplaceAll(string(*d), fmt.Sprintf("{%s}", k), fmt.Sprintf("%s", v)))
	}

	return nil
}

func fetchURL(cache *sync.Map, path string) ([]byte, error) {
	if v, ok := cache.Load(path); ok {
		klog.V(5).Infof("Retreiving from cache: %s", path)
		return v.([]byte), nil
	}
	klog.V(5).Infof("Not cached, downloading %s", path)
	resp, err := http.Get(path)
	if err != nil {
		return []byte{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return []byte{}, fmt.Errorf("%d", resp.StatusCode)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	cache.Store(path, body)
	return body, nil
}

func unMarshalTemplate(body []byte, t *templateapiv1.Template) error {
	body, err := yaml.YAMLToJSON(body)
	if err != nil {
		return fmt.Errorf("Unable to convert yaml to json: %v", err)
	}
	if jerr := json.Unmarshal(body, &t); jerr != nil {
		return fmt.Errorf("Error unmarshaling data: %v", err)

	}
	return nil
}

func unMarshalImageStream(body []byte, is *imageapiv1.ImageStream) error {
	body, err := yaml.YAMLToJSON(body)
	if err != nil {
		return fmt.Errorf("Unable to convert yaml to json: %v", err)
	}
	if jerr := json.Unmarshal(body, &is); jerr != nil {
		return fmt.Errorf("Error unmarshaling data: %v", err)

	}
	return nil
}

func unMarshalImageStreamList(body []byte, isl *imageapiv1.ImageStreamList) error {
	body, err := yaml.YAMLToJSON(body)
	if err != nil {
		return fmt.Errorf("Unable to convert yaml to json: %v", err)
	}
	if jerr := json.Unmarshal(body, &isl); jerr != nil {
		return fmt.Errorf("Error unmarshaling data: %v", err)

	}
	return nil
}

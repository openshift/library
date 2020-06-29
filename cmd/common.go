package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/ghodss/yaml"

	"k8s.io/klog"

	imageapiv1 "github.com/openshift/api/image/v1"
	templateapiv1 "github.com/openshift/api/template/v1"
)

func writeToFile(document string, data []byte, filePath string) error {
	klog.V(5).Infof("[%s] Writing file %s", document, filePath)
	if _, err := os.Stat(filepath.Dir(filePath)); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
			return fmt.Errorf("Error creating directory %q: %v", filepath.Dir(filePath), err)
		}
	}
	return ioutil.WriteFile(filePath, data, os.ModePerm)
}

func replaceVariables(document string, d *[]byte, v map[string]string) error {
	for k, v := range v {
		klog.V(5).Infof("[%s] Replacing variable {%s} with %s", document, k, v)
		*d = []byte(strings.ReplaceAll(string(*d), fmt.Sprintf("{%s}", k), fmt.Sprintf("%s", v)))
	}

	return nil
}

func fetchURL(document string, path string) ([]byte, error) {
	klog.V(5).Infof("[%s] Fetching %s", document, path)
	resp, err := http.Get(path)
	if err != nil {
		return []byte{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return []byte{}, fmt.Errorf("%d", resp.StatusCode)
	}

	return ioutil.ReadAll(resp.Body)
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

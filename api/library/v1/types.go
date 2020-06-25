package v1

// ItemImageStream is an object that describes an OpenShift ImageStream that we need to import
type ItemImageStream struct {
	Location string   `yaml:"location"`
	Docs     string   `yaml:"docs"`
	Regex    string   `yaml:"regex"`
	Suffix   string   `yaml:"suffix"`
	Tags     []string `yaml:"tags"`
}

// ItemTemplate is an object that describes an OpenShift Template that we need to import
type ItemTemplate struct {
	Location string   `yaml:"location"`
	Docs     string   `yaml:"docs"`
	Regex    string   `yaml:"regex"`
	Suffix   string   `yaml:"suffix"`
	Tags     []string `yaml:"tags"`
}

// Item is a container that holds ItemImageStreams and ItemTemplates
type Item struct {
	ImageStreams []ItemImageStream `yaml:"imagestreams"`
	Templates    []ItemTemplate    `yaml:"templates"`
}

type Document struct {
	Variables map[string]string
}

// DocumentData is the data section of a library document
type DocumentData struct {
	Data map[string]Item
}

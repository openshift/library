# OpenShift Image Streams and Templates

This repository contains a curated set of image streams and templates for OpenShift. See the official OpenShift documentation for more information about **[image streams](https://docs.okd.io/latest/openshift_images/image-streams-manage.html)** and **[templates](https://docs.okd.io/latest/openshift_images/using-templates.html)**.


- [Overview](#overview)
    - [Official](#official)
    - [Community](#community)
- [Building the Library](#building-the-library)
    - [Running the Script](#running-the-script)
    - [Verifying Your Updates](#verifying-your-updates)
- [Contributing](#contributing)
    - [YAML File Structure](#yaml-file-structure)
        - [Variables](#variables)
        - [Organization](#organization)
        - [folder_name](#folder_name)
        - [location](#location)
        - [docs](#docs)
        - [regex](#regex)
        - [suffix](#suffix)
    - [Adding Your Template or ImageStream](#adding-your-template-or-imagestream)
- [Additional Information](#additional-information)


## Overview

### Official

Provided and supported by Red Hat, official Templates and ImageStreams are listed in the top level of this repository, making it easy for developers to get started creating applications with the newest technologies.

You can check to see which of the official Templates and ImageStreams are available in your OpenShift cluster by doing one of the following:

- Log into the web console and click **Add to Project**
- List them for the openshift project using the **Command Line Interface**

    $ oc get templates -n openshift  
    $ oc get imagestreams -n openshift

### Community

Community templates and image streams are **not** provided or supported by Red Hat. This curated list of community maintained resources exemplify OpenShift best practices and provide clear documentation to serve as a reference for other developers.

## Building the Library

You must build the library executable before you can run the import.

    $ make build

### Running the Script
    # Imports the official.yaml and community.yaml without any
    # additional flags or filters
    $ make import
    
    # Increase the log level
    # Supported levels: 0,2,5,8
    $ make import LOGLEVEL=2

    # Imports the templates and imagestreams into some_dir
    $ make import DIR=some_dir

    # Imports only the foo.yaml and bar.yaml documents
    $ make import DOCUMENTS=foo.yaml,bar.yaml

    # Imports only the templates and imagestreams tagged with
    # tag1 OR tag2
    $ make import TAGS=tag1,tag2

    # Imports only the templates and imagestreams tagged with
    # tag1 AND tag2
    $ make import TAGS=tag1,tag2 MATCHALL=true
    
## Verifying Your Updates

    $ make verify
    
The `make verify` command runs the following checks:
 - verifies the Go syntax *(using gofmt)* 
 - verifies that make import has been run

## Contributing

### Adding Your Template or ImageStream

- Fork the [openshift/library](https://github.com/openshift/library) repository on github
- Add your template or image stream to the **community.yaml** or **official.yaml** file in the top level of this project
- Run the `make import` command and make sure that your template(s) and/or image-stream(s) are processed and written to the correct directory under the **community** or **official** folder and that no errors have occurred.
- Run the `make verify` command and ensure that no errors occur
- Commit and push your changes to your fork of the github repository
  - Make sure to commit any changes in the **community** and **official** folders
- Create a pull request against the [openshift/library](https://github.com/openshift/library) upstream repository

That's it!  Your pull request will be reviewed by a member of the OpenShift Team and merged if everything looks good.


### YAML file structure:

    variables: # (optional) top level block item
      <variable_name>: <value> # (optional)
    data: # (required) top level block item
      <folder_name>: # (required) folder that the below items will be stored in
        imagestreams: # (optional) list of image-streams to process into the above folder
          - location: # (required) github url to a json file or folder of json files
            regex: # (optional) matched against ['metadata']['name'] in the json file
            suffix: # (optional) suffix for the file that is created ex: ruby-<suffix>.json
            docs: # (optional) web address of the documentation for this image-stream
        templates: # (optional) list of templates to process into the above folder
          - location: # (required) github url to a template or folder of templates in json format
            regex: # (optional) matched against ['metadata']['name'] in the json file
            suffix: # (optional) suffix for the file that is created ex: ruby-<suffix>.json
            docs: # (optional) web address of the documentation for this template

#### Variables

Anything under the **data** block can contain a reference to a variable by using the following syntax:

    {variable_name}

You must also specify a value for that variable name under the **variables** block with the following syntax:

    <variable_name>: <value>

#### Organization

Listings in the **official.yaml** file will be created in a sub folder of the  **official** top level folder.  Listings in the **community.yaml** file will be created in a sub folder of the **community** top level folder.

#### folder_name

The **folder_name** is a sub folder which represents a logical grouping for a set of templates or image-streams in the top level **official** or **community** folders.

#### location

The **location** must be a publicly available url that points to either a template, image-stream, or image-stream list file in JSON or YAML format

#### docs

The **docs** is a field to list the web address of the documentation for the template, image-stream, or image-stream list

#### regex

The **regex** is a plain string that is matched against the `['metadata']['name']` element in the template or image-stream.  Make sure that the **regex** string that you provide is descriptive enough to only match the `['metadata']['name']` that you are trying to target.

#### suffix

The **suffix** is applied to the end of the filename that is created right before the .json file extension and can contain dashes (-) or underscores (_).


## Additional information

### Creating templates, image-streams, and image-stream lists

You can find more information about creating templates and image-streams in the official [OpenShift Documentation](https://docs.okd.io/latest).  Below are some quick links to important sections:

- [Writing Templates](https://docs.okd.io/latest/openshift_images/using-templates.html#templates-writing_using-templates)
- [Quickstart Templates](https://docs.okd.io/latest/openshift_images/using-templates.html#templates-using-instant-app-quickstart_using-templates)
- [Image Streams](https://docs.okd.io/latest/openshift_images/image-streams-manage.html)
- [Managing Images](https://docs.okd.io/latest/openshift_images/managing_images/managing-images-overview.html)

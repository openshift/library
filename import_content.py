#!/usr/bin/env python

"""Imports templates and imagestreams from various sources"""

import argparse
import json
import os
import re
import shutil
import requests
import yaml


SVARS = {
    "base_dir": os.path.dirname(os.path.dirname(os.path.realpath(__file__))),
    "sources": ['official', 'community'],
    "tags": [],
    "vars": {},
    "index": {}
    }

def message(action, sub_action, text):
    """ Outputs a formatted message to stdout

    Args:
        action (string): the main action that is being performed
        sub_action (string): the type of action being performed
        text (string): additional information for the user

    """
    print('{:<10} | {:<17} | {}'.format(action, sub_action, text))


def is_json(data):
    """ Checks if the data is json

    Args:
        data: The data to check

    Returns:
        bool: True if the data is valid json, False otherwise
    """
    try:
        json.loads(json.dumps(data))
    except:
        return False
    return True


def is_yaml(data):
    """ Checks if the data is valid yaml

    Args:
        data: The data to check

    Returns:
        bool: True if the data is valid yaml, False otherwise
    """
    if data[0] == "{":
        return False
    try:
        yaml.load(data)
    except:
        return False
    return True


def is_valid(data):
    """Checks the validity of a JSON or YAML document

    Args:
        data (string): the data to check

    Returns:
        bool: True if the data is valid, False otherwise
        data: None if validity is False, otherwise a Python dict object

    """
    if is_yaml(data):
        return True, yaml.load(data)
    elif is_json(data):
        data = data.replace("\t", " ")
        return True, json.loads(data)
    else:
        return False, None


def append_to_index(source, folder, sub_folder, data):
    """ Appends information about a template or image-stream to the index file
        that users can script against to find the templates and image-streams listed
        in this library

    Args:
        source (string): official|community
        folder (string): the folder that the data is located in
        sub_folder (string): templates|imagestreams
        data (dictionary): data that should be included in the listing

    """
    if source not in SVARS['index']:
        SVARS['index'][source] = {}
    if folder not in SVARS['index'][source]:
        SVARS['index'][source][folder] = {}
    if sub_folder not in SVARS['index'][source][folder]:
        SVARS['index'][source][folder][sub_folder] = []
    SVARS['index'][source][folder][sub_folder].append(data)


def replace_variables(string):
    """ Replaces the {variables} in a YAML document based on the data
        found in the variables section of the document

    Args:
        string (string): the string to replace the variables in

    Returns:
        string: the string with all of the variables replaced

    """
    matches = re.search(r'(\{[a-zA-Z0-9-_]+\})', string)
    if matches:
        for match in matches.groups():
            string = string.replace(match, SVARS['vars'][re.sub(r'\{|\}', '', match)])
            message('Processing', 'variables', 'replacing ' + match + ' with ' + SVARS['vars'][re.sub(r'\{|\}', '', match)])
    return string


def fetch_url(path):
    """ Fetches json or yaml data from a url

    Args:
        path (string): the github url to use for the API call

    Returns:
        status_code: The HTTP status code from the response
        Python dictionary: the data returned from the successful API call, or None if !200

    """
    message('Fetching', 'data', path)
    req = requests.get(path)
    if req.status_code == 200:
        message('Caching', '', path)
        valid, dict_data = is_valid(req.text)
        if not valid:
            message('Error', 'invalid data', path)
            return "Error, invalid data detected", None
        return req.status_code, dict_data
    else:
        return req.status_code, None


def write_data_to_file(data, path):
    """ Writes formatted json data to a file at path

    Args:
        data (json): json data to write to the file
        path (string): the path to write the file to, includes folder and file name

    """
    message('Writing', 'data to file', path)
    target_file = file(path, 'w')
    target_file.write(json.dumps(data, sort_keys=True, indent=4, separators=(',', ': ')))
    target_file.close()

def check_file_name(path, file_name, extension):
    """ Check if the file already exists in the directory and return an valid file name

    Args:
        path (string): the path to write the file to, includes folder and file name
        file_name (string): the file name
        extension (string): the name of file extension (such as .json)

    Returns:
        file_name (string): The valid file name that can be written into

    """
    i = 1
    name = file_name
    while os.path.exists(path + name + extension):
        name = file_name + "_" + str(i)
        i = i + 1
    return name


def process_template(source, folder, location_list, template):
    """ Processes a template and writes it's data to a file in JSON format

    Args:
        source (string): item from the sources list
        folder (string): the folder to place the file in
        location_list (dictionary): information about the template from the YAML file
        template (json): information about the data to get for the template

    """
    message('Processing', 'template', template['metadata']['name'])
    matches = False
    if 'regex' in location_list:
        matches = re.search(r'(^' + location_list['regex'] + ')', template["metadata"]["name"])
    if matches or 'regex' not in location_list:
        file_name = check_file_name(SVARS['base_dir'] + '/' + folder + '/templates/', template['metadata']['name'], '.json')
        index_data = {
            'name': template['metadata']['name'],
            'docs': location_list['docs'] if 'docs' in location_list else '',
            'source_url': location_list['location'],
            'description': template['metadata']['annotations']['description'] if 'description' in template['metadata']['annotations'] else '',
            'path': (source + '/' if source in SVARS["sources"] else '') + folder + '/templates/' + file_name + '.json'
            }
        append_to_index(source, folder, 'templates', index_data)
        write_data_to_file(template, SVARS['base_dir'] + '/' + folder + '/templates/' + file_name + '.json')


def process_imagestream(source, folder, location_list, imagestream):
    """ Processes an image-stream and writes it's data to a file in JSON format

    Args:
        source (string): item from the sources list
        folder (string): the folder to place the file in
        location_list (dictionary): information about the image-stream from the YAML file
        imagestream (json): information about the data to get for the image-stream

    """
    message('Processing', 'image-stream', '')
    imagestreams = []
    if imagestream['kind'] == 'ImageStream':
        message('Found', 'image-stream', location_list['location'])
        imagestreams.append(imagestream)
    elif imagestream['kind'] == 'ImageStreamList' or imagestream['kind'] == 'List':
        message('Found', 'image-stream list', location_list['location'])
        for istream in imagestream['items']:
            imagestreams.append(istream)
    for stream in imagestreams:
        matches = False
        if 'regex' in location_list:
            matches = re.search(r'(^' + location_list['regex'] + ')', stream["metadata"]["name"])
        if matches or 'regex' not in location_list:
            file_name = check_file_name(SVARS['base_dir'] + "/" + folder + "/imagestreams/", \
                                stream["metadata"]["name"] + ("-" + location_list["suffix"] if 'suffix' in location_list else ''), '.json')
            index_data = {
                'name': stream['metadata']['name'],
                'docs': location_list['docs'] if 'docs' in location_list else '',
                'source_url': location_list['location'],
                'path': (source + '/' if source in SVARS["sources"] else '') + folder + '/imagestreams/' + file_name + ".json"
                }
            append_to_index(source, folder, 'imagestreams', index_data)
            write_data_to_file(stream, SVARS['base_dir'] + "/" + folder + "/imagestreams/" + file_name + '.json')


def create_indexes():
    """ Creates the index.json and README.md files """

    for source in SVARS['tags']:
        write_data_to_file(SVARS['index'][source], source + '/index.json')

        with open(source + '/README.md', 'a') as index_file:
            for folder, sections in sorted(SVARS['index'][source].items()):
                index_file.write('# ' + folder + '\n')
                for section, items in sorted(sections.items()):
                    index_file.write('## ' + section + '\n')
                    for item in items:
                        index_file.write('### ' + item['name'] + '\n')
                        index_file.write('Source URL: [' + item['source_url'] + '](' + item['source_url'] + ' )  \n')
                        if 'docs' in item and item['docs'] != '':
                            index_file.write('Docs: [' + item['docs'] + '](' + item['docs'] + ')  \n')
                        index_file.write('Path: ' + item['path'] + '  \n')


def main():
    """ Runs the main program, gets the data from the YAML file(s)
        and fires off the functions defined above

    """
    # parse command line options
    parser = argparse.ArgumentParser(description='Build OpenShift template and image-stream library')
    parser.add_argument("-t", "--tags", nargs='?', help="Select specific tag(s) to import templates/imagestreams (separated by comma ',')")
    args = parser.parse_args()

    if not os.path.exists('tmp'):
        os.makedirs('tmp')

    # Get custom tags information and add to SVARS["tags"]
    # Otherwise, use the default tags as community and official
    tags = args.tags
    if tags:
        SVARS['tags'] = tags.split(",")
        # Delete the custom tag(s) directori(es)
        for tag in SVARS['tags']:
            if os.path.exists(tag):
                message("Deleting", "folder", tag)
                shutil.rmtree(tag)
    else:
        SVARS['tags'] = SVARS['sources']

    for source in SVARS['sources']:
        message('Opening', 'source file', source + '.yaml')
        with open(source + '.yaml', 'r') as source_file:
            if not tags:
                SVARS['base_dir'] = os.path.dirname(os.path.realpath(__file__)) + '/' + source
            else:
                SVARS['base_dir'] = os.path.dirname(os.path.realpath(__file__))
            message('Opening', 'file path', SVARS['base_dir'])
            raw_yaml = source_file.read()
            valid, doc = is_valid(raw_yaml)
            if not valid:
                message('Error', 'file', 'unable to load file ' + args.source)
                exit(1)
            if 'variables' in doc:
                for key, val in doc['variables'].items():
                    SVARS['vars'][key] = val
            else:
                message('Info', '', 'No variables found in source document')

            valid, doc = is_valid(replace_variables(raw_yaml))
            if not valid:
                message('Error', 'YAML', 'Variable replacement caused invalid YAML')
                exit(1)
        if 'data' in doc:
            if os.path.exists(source):
                message("Deleting", "folder", source)
                shutil.rmtree(source)
            for folder, contents in doc['data'].items():
                if not tags:
                    if not os.path.exists(os.path.join(SVARS['base_dir'], folder)):
                        os.makedirs(os.path.join(SVARS['base_dir'], folder))
                for item_type in ['imagestreams', 'templates']:
                    if item_type in contents and len(contents[item_type]) > 0:
                        for item in contents[item_type]:
                            # Check if custom tags are provided:
                            if tags:
                                # Check for the "openshift" tag in YAML file
                                if "openshift" in item:
                                    for tag in SVARS["tags"]:
                                        if not os.path.exists(os.path.join(SVARS['base_dir'], tag)):
                                            os.makedirs(os.path.join(SVARS['base_dir'], tag))
                                        status, dict_data = fetch_url(item['location'])
                                        if status != 200:
                                            message('Error', 'Not Found', item['location'])
                                            exit(1)
                                        options = item.get("openshift")
                                        # Import imagestreams/templates for the option
                                        if tag in options:
                                            if item_type == 'templates':
                                                if not os.path.exists(os.path.join(SVARS['base_dir'], tag, "templates")):
                                                    os.makedirs(os.path.join(SVARS['base_dir'], tag, "templates"))
                                                process_template(tag, tag, item, dict_data)
                                            elif item_type == 'imagestreams':
                                                if not os.path.exists(os.path.join(SVARS['base_dir'], tag, "imagestreams")):
                                                    os.makedirs(os.path.join(SVARS['base_dir'], tag, "imagestreams"))
                                                process_imagestream(tag, tag, item, dict_data)
                                else:
                                    message("Skip", "No 'openshift' tag found for " + item_type, item["location"])
                            # If no custom tags are provided
                            else:
                                if not os.path.exists(os.path.join(SVARS['base_dir'], folder, item_type)):
                                    os.makedirs(os.path.join(SVARS['base_dir'], folder, item_type))
                                status, dict_data = fetch_url(item['location'])
                                if item_type == 'templates':
                                    process_template(source, folder, item, dict_data)
                                elif item_type == 'imagestreams':
                                    process_imagestream(source, folder, item, dict_data)
    create_indexes()

if __name__ == '__main__':
    main()

#!/usr/bin/env python3

"""Imports templates and imagestreams from various sources"""

import argparse
import hashlib
import json
import os
import re
import shutil
import requests
import yaml

SVARS = {
    "base_dir"      : os.path.dirname(os.path.dirname(os.path.realpath(__file__))),
    "sources"       : ['official', 'community'],
    "vars"          : {},
    "index"         : {}
    }

def message(action, sub_action, text):
    """ Outputs a formatted message to stdout

    Args:
        action (string): the main action that is being performed
        sub_action (string): the type of action being performed
        text (string): additional information for the user

    """
    print('{:<10} | {:<17} | {}'.format(action, sub_action, text))

def is_valid(data, data_type):
    """Checks the validity of a JSON or YAML document

    Args:
        data (string): the data to check
        data_type (string): json|yaml

    Returns:
        bool: True if the data is valid, False otherwise
        data: None if validity is False, otherwise a JSON or YAML object

    """
    try:
        if data_type == 'json':
            loaded_data = json.loads(json.dumps(data))
        elif data_type == 'yaml':
            loaded_data = yaml.load(data)
    except:
        message('Error', 'validation', 'data is not valid ' + data_type)
        return False, None
    message('Success', 'validation', 'data is valid ' + data_type)
    return True, loaded_data

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
    if not source in SVARS['index']:
        SVARS['index'][source] = {}
    if not folder in SVARS['index'][source]:
        SVARS['index'][source][folder] = {}
    if not sub_folder in SVARS['index'][source][folder]:
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

def fetch_or_retrieve_url(path):
    """ Fetches json data from a url and caches it locally for faster access

    Args:
        path (string): the github url to use for the API call

    Returns:
        status_code: None if successful, req.status_code if !200
        JSON object: the JSON returned from the successful API call, or None if !200

    """
    path_hash = hashlib.md5()
    path_hash.update(str.encode(path))
    cache_path = 'tmp/' + path_hash.hexdigest()
    if os.path.exists(cache_path) and SVARS['cache']:
        message('Retrieving', 'from cache', path)
        with open(cache_path, 'r') as cached_file:
            valid, json_data = is_valid(json.loads(cached_file.read()), 'json')
            if not valid:
                return "Error, invalid json detected", None
            return None, json_data
    else:
        message('Fetching', 'data', path)
        req = requests.get(path)
        if req.status_code == 200:
            message('Caching', '', path)
            valid, json_data = is_valid(req.json(), 'json')
            if not valid:
                message('Error', 'invalid JSON', path)
                return "Error, invalid json detected", None
            write_data_to_file(json_data, cache_path)
            return None, json_data
        else:
            return req.status_code, None

def write_data_to_file(data, path):
    """ Writes data to a file at path

    Args:
        data (json): json data to write to the file
        path (string): the path to write the file to, includes folder and file name

    """
    message('Writing', 'data to file', path)
    with open(path, 'wb') as target_file:
        target_file.write(bytes(json.dumps(data, sort_keys=True, indent=4), 'utf-8'))

def process_template(source, folder, location_list, template):
    """ Processes a template and writes it's data to a file in JSON format

    Args:
        folder (string): the folder to place the file in
        location_list (dictionary): information about the template from the YAML file
        template (json): information about the data to get for the template

    """
    message('Processing', 'template', template['metadata']['name'])
    matches = False
    if 'regex' in location_list:
        if valid:
            matches = re.search(r'(^' + location_list['regex'] + ')', template["metadata"]["name"])
    if matches or 'regex' not in location_list:
        index_data = {
            'name': template['metadata']['name'],
            'docs': location_list['docs'] if 'docs' in location_list else '',
            'source_url': location_list['location'],
            'description': template['metadata']['annotations']['description'] if 'description' in template['metadata']['annotations'] else '',
            'path': source + '/' + folder + '/templates/' + template['metadata']['name'] + '.json'
            }
        append_to_index(source, folder, 'templates', index_data)
        write_data_to_file(template, SVARS['base_dir'] + '/' + folder + '/templates/' + template['metadata']['name'] + '.json')

def process_imagestream(source, folder, location_list, imagestream):
    """ Processes an image-stream and writes it's data to a file in JSON format

    Args:
        folder (string): the folder to place the file in
        location_list (dictionary): information about the image-stream from the YAML file
        imagestream (json): information abou the data to get for the image-stream

    """
    message('Processing', 'image-stream', 'test')
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
            index_data = {
                'name': stream['metadata']['name'],
                'docs': location_list['docs'] if 'docs' in location_list else '',
                'source_url': location_list['location'],
                'path': source + '/' + folder + '/imagestreams/' +  stream["metadata"]["name"] + ("-" + location_list["suffix"] if 'suffix' in location_list else '') + ".json"
                }
            append_to_index(source, folder, 'imagestreams', index_data)
            write_data_to_file(stream, SVARS['base_dir'] + "/" + folder + "/imagestreams/" + stream["metadata"]["name"] + ("-" + location_list["suffix"] if 'suffix' in location_list else '') + ".json")

def create_indexes():
    """ Creates the index.json and index.md files """

    for source in SVARS['sources']:
        write_data_to_file(SVARS['index'][source], source + '/index.json')

        with open(source + '/index.md', 'a') as index_file:
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
    parser.add_argument('--no-clean', dest='clean', action='store_false', help='Do not remove the tmp directory')
    parser.add_argument('--no-cache', dest='cache', action='store_false', help='Disable caching of github api requests')
    args = parser.parse_args()

    SVARS['cache'] = args.cache
    SVARS['clean'] = args.clean

    os.makedirs('tmp', exist_ok=True)

    for source in SVARS['sources']:
        message('Opening', 'source file', source + '.yaml')
        with open(source + '.yaml', 'r') as source_file:
            SVARS['base_dir'] = os.path.dirname(os.path.realpath(__file__)) + '/' + source
            message('Opening', 'file path', SVARS['base_dir'])
            raw_yaml = source_file.read()
            valid, doc = is_valid(raw_yaml, 'yaml')
            if not valid:
                message('Error', 'file', 'unable to load file ' + args.source)
                exit(1)
            if 'variables' in doc:
                for key, val in doc['variables'].items():
                    SVARS['vars'][key] = val
            else:
                message('Info', '', 'No variables found in source document')

            valid, doc = is_valid(replace_variables(raw_yaml), 'yaml')
            if not valid:
                message('Error', 'YAML', 'Variable replacement caused invalid YAML')
                exit(1)
        if 'data' in doc:
            if os.path.exists(source):
                message("Deleting", "folder", source)
                shutil.rmtree(source)
            for folder, contents in doc['data'].items():
                if not os.path.exists(os.path.join(SVARS['base_dir'], folder)):
                    os.makedirs(os.path.join(SVARS['base_dir'], folder))
                for item_type in ['imagestreams', 'templates']:
                    if item_type in contents and len(contents[item_type]) > 0:
                        for item in contents[item_type]:
                            if not os.path.exists(os.path.join(SVARS['base_dir'], folder, item_type)):
                                os.makedirs(os.path.join(SVARS['base_dir'], folder, item_type))
                            status, json_data = fetch_or_retrieve_url(item['location'])
                            if item_type == 'templates':
                                process_template(source, folder, item, json_data)
                            elif item_type == 'imagestreams':
                                process_imagestream(source, folder, item, json_data)
    create_indexes()
    if SVARS['clean'] and os.path.exists('tmp'):
        message('Removing', 'directory', 'tmp')
        shutil.rmtree('tmp')

if __name__ == '__main__':
    main()

import os
import xml.etree.ElementTree as ET
import re

def print_elements(element, parent_path="", seen_paths={}, element_samples={}, file_occurrences={}, current_filename=""):
    tag = element.tag[element.tag.find("}")+1:] if "}" in element.tag else element.tag
    current_path = f"{parent_path}/{tag}" if parent_path else tag
    getty_pattern = re.compile(r"http://vocab.getty.edu/page/aat/\d+")
    
    # Update occurrences for elements within a single file
    file_occurrences[current_path] = file_occurrences.get(current_path, 0) + 1

    # Handle element sample value
    if current_path not in element_samples and element.text and element.text.strip():
        element_samples[current_path] = {
            'value': element.text.strip(),
            'filename': current_filename
        }

    # Update global occurrences for elements
    if current_path in seen_paths:
        seen_paths[current_path]['occurrences'] += 1
    else:
        seen_paths[current_path] = {
            'occurrences': 1,
            'max_per_file': 0,
            'max_file': '',
            'filename': current_filename,
        }

    # Handle attributes
    for attr, value in element.attrib.items():
        # collapse all the getty URIs into one
        if "http://vocab.getty.edu/page/aat" in value:
            value = re.sub(getty_pattern, "http://vocab.getty.edu/page/aat/*", value)
        attr_path = f"{current_path}/@{attr}/{value}"
        file_occurrences[attr_path] = file_occurrences.get(attr_path, 0) + 1
        if attr_path not in seen_paths:
            if element.text and element.text.strip():
                seen_paths[attr_path] = {
                    'occurrences': 1,
                    'max_per_file': 0,
                    'max_file': '',
                }
                element_samples[attr_path] = {
                    'value': element.text.strip(),
                    'filename': current_filename,
                }
        else:
            seen_paths[attr_path]['occurrences'] += 1

    for child in element:
        print_elements(child, current_path, seen_paths, element_samples, file_occurrences, current_filename)

def update_max_per_file(seen_paths, file_occurrences, filename):
    for path, count in file_occurrences.items():
        if path in seen_paths and count > seen_paths[path]['max_per_file']:
            seen_paths[path]['max_per_file'] = count
            seen_paths[path]['max_file'] = filename

def process_xml_folders(folders):
    seen_paths = {}
    element_samples = {}
    for folder in folders:
        for filename in os.listdir(folder):
            if filename.endswith('.xml'):
                path = os.path.join(folder, filename)
                file_occurrences = {}  # Reset for each file
                try:
                    tree = ET.parse(path)
                    root = tree.getroot()
                    current_filename = os.path.basename(filename).replace(".xml", "")
                    print_elements(root, seen_paths=seen_paths, element_samples=element_samples, file_occurrences=file_occurrences, current_filename=current_filename)
                    update_max_per_file(seen_paths, file_occurrences, current_filename)
                except ET.ParseError as e:
                    print(f"Error parsing {filename}: {e}")

    # After processing all files, print paths, sample values, occurrences, max occurrences per file, and filename of max occurrence
    for path, data in seen_paths.items():
        sample_value = element_samples.get(path, {'value': "N/A", 'filename': "N/A"})['value']
        sample_filename = element_samples.get(path, {'value': "N/A", 'filename': "N/A"})['filename']
        print(f"{path}\t{sample_value}\t{data['occurrences']}\t{data['max_per_file']}\t{sample_filename}")

folders = ['../001-extract-mods/xml/digitalcollections', '../001-extract-mods/xml/preserve']
process_xml_folders(folders)

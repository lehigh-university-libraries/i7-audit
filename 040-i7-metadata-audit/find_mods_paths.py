import os
import xml.etree.ElementTree as ET

def print_elements(element, parent_path="", seen_paths={}, element_samples={}, file_occurrences={}):
    tag = element.tag[element.tag.find("}")+1:] if "}" in element.tag else element.tag
    current_path = f"{parent_path}/{tag}" if parent_path else tag
    
    # Update occurrences for elements within a single file
    file_occurrences[current_path] = file_occurrences.get(current_path, 0) + 1

    # Handle element sample value
    if current_path not in element_samples and element.text and element.text.strip():
        element_samples[current_path] = element.text.strip()

    # Update global occurrences for elements
    if current_path in seen_paths:
        seen_paths[current_path]['occurrences'] += 1
    else:
        seen_paths[current_path] = {'occurrences': 1, 'max_per_file': 0}

    # Handle attributes
    for attr, value in element.attrib.items():
        attr_path = f"{current_path}/@{attr}/{value}"
        file_occurrences[attr_path] = file_occurrences.get(attr_path, 0) + 1
        if attr_path not in seen_paths:
            seen_paths[attr_path] = {'occurrences': 1, 'max_per_file': 0}
        else:
            seen_paths[attr_path]['occurrences'] += 1

    for child in element:
        print_elements(child, current_path, seen_paths, element_samples, file_occurrences)

def update_max_per_file(seen_paths, file_occurrences):
    for path, count in file_occurrences.items():
        if path in seen_paths:
            seen_paths[path]['max_per_file'] = max(seen_paths[path]['max_per_file'], count)

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
                    print_elements(root, seen_paths=seen_paths, element_samples=element_samples, file_occurrences=file_occurrences)
                    update_max_per_file(seen_paths, file_occurrences)
                except ET.ParseError as e:
                    print(f"Error parsing {filename}: {e}")

    # After processing all files, print paths, sample values, occurrences, and max occurrences per file
    for path, data in seen_paths.items():
        sample_value = element_samples.get(path, "N/A")
        print(f"{path}\t{sample_value}\t{data['occurrences']}\t{data['max_per_file']}")

folders = ['../001-extract-mods/xml/digitalcollections', '../001-extract-mods/xml/preserve']
process_xml_folders(folders)

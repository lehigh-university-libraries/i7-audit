import os
import xml.etree.ElementTree as ET

def print_elements(element, parent_path="", seen_paths={}, element_samples={}):
    tag = element.tag[element.tag.find("}")+1:] if "}" in element.tag else element.tag
    current_path = f"{parent_path}/{tag}" if parent_path else tag
    
    # Handle element sample value
    if current_path not in element_samples and element.text and element.text.strip():
        element_samples[current_path] = element.text.strip()

    # Update occurrences for elements
    if current_path in seen_paths:
        seen_paths[current_path]['occurrences'] += 1
    else:
        seen_paths[current_path] = {'occurrences': 1}

    # Handle attributes
    for attr, value in element.attrib.items():
        attr_path = f"{current_path}/@{attr}/{value}"
        if attr_path not in seen_paths:
            seen_paths[attr_path] = {'occurrences': 1}
        else:
            seen_paths[attr_path]['occurrences'] += 1

    for child in element:
        print_elements(child, current_path, seen_paths, element_samples)

def process_xml_folders(folders):
    seen_paths = {}
    element_samples = {}
    for folder in folders:
        for filename in os.listdir(folder):
            if filename.endswith('.xml'):
                path = os.path.join(folder, filename)
                try:
                    tree = ET.parse(path)
                    root = tree.getroot()
                    print_elements(root, seen_paths=seen_paths, element_samples=element_samples)
                except ET.ParseError as e:
                    print(f"Error parsing {filename}: {e}")

    # After processing all files, print paths, sample values, and occurrences
    for path, data in seen_paths.items():
        if '/@' in path:  # It's an attribute value
            print(f"{path}\t{data['occurrences']}")
        else:  # It's an element
            sample_value = element_samples.get(path, "N/A")
            print(f"{path}\t{sample_value}\t{data['occurrences']}")

folders = ['../001-extract-mods/xml/digitalcollections', '../001-extract-mods/xml/preserve']
process_xml_folders(folders)

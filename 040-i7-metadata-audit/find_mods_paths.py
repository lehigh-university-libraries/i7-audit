import os
import xml.etree.ElementTree as ET

def print_elements(element, parent_path="", seen_paths={}):
    tag = element.tag[element.tag.find("}")+1:] if "}" in element.tag else element.tag
    current_path = f"{parent_path}/{tag}" if parent_path else tag
    text = element.text.strip() if element.text and element.text.strip() else "N/A"
    
    # Update occurrences count and set sample value if it's the first occurrence
    if current_path not in seen_paths:
        seen_paths[current_path] = {'sample': text, 'occurrences': 1}
    else:
        seen_paths[current_path]['occurrences'] += 1

    for attr, value in element.attrib.items():
        attr_path = f"{current_path}/@{attr}"
        if attr_path not in seen_paths:
            seen_paths[attr_path] = {'sample': value, 'occurrences': 1}
        else:
            seen_paths[attr_path]['occurrences'] += 1

    for child in element:
        print_elements(child, current_path, seen_paths)

def process_xml_folders(folders):
    seen_paths = {}
    for folder in folders:
        for filename in os.listdir(folder):
            if filename.endswith('.xml'):
                path = os.path.join(folder, filename)
                try:
                    tree = ET.parse(path)
                    root = tree.getroot()
                    print_elements(root, seen_paths=seen_paths)
                except ET.ParseError as e:
                    print(f"Error parsing {filename}: {e}")

    # After processing all files, print paths, sample values, and occurrences
    for path, data in seen_paths.items():
        print(f"{path}\t{data['sample']}\t{data['occurrences']}")

folders = ['output/xml/digitalcollections', 'output/xml/preserve']
process_xml_folders(folders)


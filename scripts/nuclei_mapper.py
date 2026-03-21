import json
import requests
import re
import os

# URLs for latest data
WAPPALYZER_URL = "https://raw.githubusercontent.com/projectdiscovery/wappalyzergo/master/fingerprints_data.json"
NUCLEI_TEMPLATES_URL = "https://api.github.com/repos/projectdiscovery/nuclei-templates/git/trees/main?recursive=1"

def get_wappalyzer_techs():
    print("[*] Fetching Wappalyzer technologies...")
    resp = requests.get(WAPPALYZER_URL)
    data = resp.json()
    # The actual PD format uses the 'apps' key
    if "apps" in data:
        return list(data["apps"].keys())
    return []

def get_nuclei_tags():
    print("[*] Fetching Nuclei template paths...")
    # This is a proxy for tags, we'll extract names from template paths
    resp = requests.get(NUCLEI_TEMPLATES_URL)
    data = resp.json()
    tags = set()
    for item in data.get("tree", []):
        path = item.get("path", "")
        if path.endswith(".yaml"):
            # Extract directory names as tags (e.g., technologies/wordpress/...)
            parts = path.split("/")
            if len(parts) > 1:
                tags.add(parts[-2])
            # Also extract from filename if it's in a relevant category
            filename = parts[-1].replace(".yaml", "")
            tags.add(filename.split("-")[0])
    return sorted(list(tags))

def generate_map(wapp_techs, nuclei_tags):
    print(f"[*] Analyzing {len(wapp_techs)} techs and {len(nuclei_tags)} potential tags...")
    mapping = {}
    
    # Simple matching logic (to be reviewed by user/AI)
    for tech in wapp_techs:
        slug = tech.lower().replace(".js", "").replace(" ", "-")
        slug = re.sub(r'[^a-z0-9\-]', '', slug)
        
        # Check if slug exists in nuclei tags
        if slug in nuclei_tags:
            mapping[tech] = slug
        elif slug.split("-")[0] in nuclei_tags:
            mapping[tech] = slug.split("-")[0]
            
    return mapping

def main():
    try:
        wapp_techs = get_wappalyzer_techs()
        nuclei_tags = get_nuclei_tags()
        
        mapping = generate_map(wapp_techs, nuclei_tags)
        
        output_file = "nuclei-map.json"
        with open(output_file, "w") as f:
            json.dump(mapping, f, indent=4)
            
        print(f"[+] Successfully generated {len(mapping)} mappings!")
        print(f"[+] File saved to: {output_file}")
        print("[!] Tip: You can now review this JSON and ask Gemini to improve specific entries.")
        
    except Exception as e:
        print(f"[!] Error: {e}")

if __name__ == "__main__":
    main()

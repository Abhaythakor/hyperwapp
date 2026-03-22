import json
import requests
import re
import os
import sys

# URLs
WAPPALYZER_URL = "https://raw.githubusercontent.com/projectdiscovery/wappalyzergo/master/fingerprints_data.json"
NUCLEI_TEMPLATES_URL = "https://api.github.com/repos/projectdiscovery/nuclei-templates/git/trees/main?recursive=1"

MAP_FILE = "nuclei-map.json"
STATE_FILE = ".mapper_state.json"

def fetch_current():
    print("[*] Fetching latest Wappalyzer and Nuclei data...")
    wapp_resp = requests.get(WAPPALYZER_URL).json()
    wapp = wapp_resp.get('apps', {}).keys()
    
    nuclei = set()
    n_data = requests.get(NUCLEI_TEMPLATES_URL).json().get('tree', [])
    for item in n_data:
        path = item.get('path', '').lower()
        if path.endswith('.yaml'):
            parts = path.split('/')
            # Folders
            for p in parts[:-1]: 
                if len(p) > 2: nuclei.add(p)
            # File ID prefix
            filename = parts[-1].replace('.yaml', '')
            prefix = filename.split('-')[0]
            if len(prefix) > 2: nuclei.add(prefix)
            
    return set(wapp), nuclei

def main():
    # 1. Load Current Map
    mapping = {}
    if os.path.exists(MAP_FILE):
        with open(MAP_FILE, "r") as f:
            mapping = json.load(f)

    # 2. Load Last State
    state = {"techs": [], "tags": []}
    if os.path.exists(STATE_FILE):
        with open(STATE_FILE, "r") as f:
            state = json.load(f)

    # 3. Fetch New Data
    current_techs, current_tags = fetch_current()

    # 4. Find Diffs (What is actually NEW)
    added_techs = current_techs - set(state["techs"])
    added_tags = current_tags - set(state["tags"])
    removed_techs = set(state["techs"]) - current_techs

    # 5. Handle AI Prompt Flag
    if "--ai-prompt" in sys.argv:
        # Find techs that are in the system but NOT in the map
        unmapped = [t for t in current_techs if t not in mapping]
        
        task = {
            "action": "AI_REVIEW_REQUEST",
            "context": "Identify mappings between these technologies and Nuclei tags.",
            "unmapped_technologies": unmapped[:100], # Limit to 100 for token safety
            "new_nuclei_tags_found": list(added_tags)[:50],
            "total_unmapped_remaining": len(unmapped)
        }
        print("\n--- COPY THIS TO GEMINI ---")
        print(json.dumps(task, indent=2))
        print("---------------------------")
        return

    # 6. Update State
    if added_techs or added_tags or removed_techs:
        print(f"[!] Changes since last sync:")
        if added_techs: print(f"    + {len(added_techs)} New Techs")
        if added_tags: print(f"    + {len(added_tags)} New Tags")
        if removed_techs: print(f"    - {len(removed_techs)} Removed Techs")
        
        with open(STATE_FILE, "w") as f:
            json.dump({"techs": list(current_techs), "tags": list(current_tags)}, f)
        print("[+] State updated.")
    else:
        print("[+] Everything is already in sync with GitHub.")

if __name__ == "__main__":
    main()

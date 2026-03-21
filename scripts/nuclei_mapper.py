import json
import requests
import re
import os
import hashlib

# URLs
WAPPALYZER_URL = "https://raw.githubusercontent.com/projectdiscovery/wappalyzergo/master/fingerprints_data.json"
NUCLEI_TEMPLATES_URL = "https://api.github.com/repos/projectdiscovery/nuclei-templates/git/trees/main?recursive=1"

MAP_FILE = "nuclei-map.json"
STATE_FILE = ".mapper_state.json"

def get_wappalyzer_techs():
    resp = requests.get(WAPPALYZER_URL)
    data = resp.json()
    return list(data.get("apps", {}).keys())

def get_nuclei_tags():
    resp = requests.get(NUCLEI_TEMPLATES_URL)
    data = resp.json()
    tags = set()
    for item in data.get("tree", []):
        path = item.get("path", "")
        if path.endswith(".yaml"):
            parts = path.split("/")
            if len(parts) > 1: tags.add(parts[-2])
            filename = parts[-1].replace(".yaml", "")
            tags.add(filename.split("-")[0])
    return sorted(list(tags))

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
    print("[*] Checking for updates...")
    current_techs = get_wappalyzer_techs()
    current_tags = get_nuclei_tags()

    # 4. Find Diffs
    new_techs = set(current_techs) - set(state["techs"])
    removed_techs = set(state["techs"]) - set(current_techs)
    new_tags = set(current_tags) - set(state["tags"])

    if not new_techs and not new_tags and not removed_techs:
        print("[+] Everything is up to date! No changes found.")
        return

    print(f"[!] Changes detected:")
    if new_techs: print(f"    - New Techs: {len(new_techs)}")
    if new_tags: print(f"    - New Tags: {len(new_tags)}")
    if removed_techs: print(f"    - Removed Techs: {len(removed_techs)}")

    # 5. Handle removals
    for tech in removed_techs:
        if tech in mapping:
            del mapping[tech]

    # 6. Attempt auto-mapping for new techs
    pending_review = []
    for tech in new_techs:
        slug = tech.lower().replace(".js", "").replace(" ", "-")
        slug = re.sub(r'[^a-z0-9\-]', '', slug)
        
        if slug in current_tags:
            mapping[tech] = slug
        elif slug.split("-")[0] in current_tags:
            mapping[tech] = slug.split("-")[0]
        else:
            pending_review.append(tech)

    # 7. Save Results
    with open(MAP_FILE, "w") as f:
        json.dump(mapping, f, indent=4, sort_keys=True)
    
    with open(STATE_FILE, "w") as f:
        json.dump({"techs": current_techs, "tags": current_tags}, f)

    print(f"\n[+] Mapping updated. Total mappings: {len(mapping)}")
    
    if pending_review:
        print(f"\n[?] {len(pending_review)} new techs need AI review.")
        print("[>] Run this command to see the list for Gemini:")
        print("    python3 scripts/nuclei_mapper.py --list-new")

if __name__ == "__main__":
    import sys
    if "--list-new" in sys.argv:
        if os.path.exists(STATE_FILE) and os.path.exists(MAP_FILE):
            with open(STATE_FILE, "r") as f: state = json.load(f)
            with open(MAP_FILE, "r") as f: mapping = json.load(f)
            new_but_unmapped = [t for t in get_wappalyzer_techs() if t not in mapping]
            print("\n--- NEW TECHS FOR AI REVIEW ---")
            print(json.dumps(new_but_unmapped, indent=2))
            print("------------------------------")
        else:
            print("[!] Run the script without flags first to establish state.")
    else:
        main()

# Offline Input Formats 📁

HyperWapp features an intelligent offline engine that can reconstruct HTTP contexts from various tool outputs. This allows you to perform technology detection on data you have already collected.

---

## 1. Supported Formats

### Katana Output (Recursive)
HyperWapp natively supports the directory structure produced by [Katana](https://github.com/projectdiscovery/katana).
*   **Detection:** Looks for folders containing `.txt` files that start with a `GET /` request line.
*   **Behavior:** It recursively walks through all subdirectories, identifying the domain from the parent folder names and reconstructing the full URL from the internal request headers.
*   **Example:** `hyperwapp -offline ./katana_responses/`

### FFF Output (Domain/Hash)
Supports the directory structure produced by [FFF](https://github.com/tomnomnom/fff).
*   **Detection:** Looks for directories where files are split into `<hash>.headers` and `.body`.
*   **Behavior:** HyperWapp automatically groups these pairs together to provide Wappalyzer with the full HTTP context (Headers + Body).
*   **Example:** `hyperwapp -offline ./fff_output/`

### Raw HTTP Dumps
Processes files containing raw HTTP response blocks.
*   **Detection:** Files containing `HTTP/1.1 200 OK` followed by standard headers.
*   **Behavior:** Splits the file into individual response objects and extracts technology fingerprints.
*   **Example:** `hyperwapp -offline ./responses.txt`

### Body-Only Files
If a directory or file doesn't match the above patterns, HyperWapp falls back to "Body-Only" mode.
*   **Behavior:** It recursively reads every file (HTML, JS, CSS, JSON) and treats the content as an HTTP body.
*   **Inference:** The domain is inferred from the filename.
*   **Example:** `hyperwapp -offline ./my_web_assets/`

---

## 2. The Auto-Detection Logic

When you run `hyperwapp -offline <path>`, the tool follows this priority:
1.  **Is it an FFF structure?** (Checks for `.headers` / `.body` pairs)
2.  **Is it a Katana structure?** (Checks for `.txt` files with request/response blocks)
3.  **Is it a Raw HTTP file?** (Peeks inside for HTTP status lines)
4.  **Fallback:** Process as recursive Body-Only targets.

---

## 3. Why Use Offline Mode?
*   **Speed:** Scanning local files is 100x faster than making network requests.
*   **Stealth:** Re-analyze your results without sending more packets to the target.
*   **Portability:** Collect data on a high-bandwidth server and analyze it on your local machine.

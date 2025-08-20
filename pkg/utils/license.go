package utils

import (
	"strings"
)

// getLicenseURL returns the canonical URL for well-known licenses
func GetLicenseURL(licenseID string) string {
	licenseID = strings.ToLower(strings.TrimSpace(licenseID))

	licenseURLs := map[string]string{
		"apache-2.0":            "https://www.apache.org/licenses/LICENSE-2.0",
		"mit":                   "https://opensource.org/licenses/MIT",
		"bsd-3-clause":          "https://opensource.org/licenses/BSD-3-Clause",
		"bsd-2-clause":          "https://opensource.org/licenses/BSD-2-Clause",
		"gpl-3.0":               "https://www.gnu.org/licenses/gpl-3.0.html",
		"gpl-2.0":               "https://www.gnu.org/licenses/old-licenses/gpl-2.0.html",
		"lgpl-3.0":              "https://www.gnu.org/licenses/lgpl-3.0.html",
		"lgpl-2.1":              "https://www.gnu.org/licenses/old-licenses/lgpl-2.1.html",
		"cc-by-4.0":             "https://creativecommons.org/licenses/by/4.0/",
		"cc-by-sa-4.0":          "https://creativecommons.org/licenses/by-sa/4.0/",
		"cc-by-nc-4.0":          "https://creativecommons.org/licenses/by-nc/4.0/",
		"cc0-1.0":               "https://creativecommons.org/publicdomain/zero/1.0/",
		"unlicense":             "https://unlicense.org/",
		"llama2":                "https://github.com/facebookresearch/llama/blob/main/LICENSE",
		"llama3":                "https://github.com/meta-llvm/llama-models/blob/main/models/llama3/LICENSE",
		"llama3.1":              "https://github.com/meta-llvm/llama-models/blob/main/models/llama3_1/LICENSE",
		"llama3.2":              "https://github.com/meta-llvm/llama-models/blob/main/models/llama3_2/LICENSE",
		"llama3.3":              "https://github.com/meta-llvm/llama-models/blob/main/models/llama3_3/LICENSE",
		"llama4":                "https://github.com/meta-llvm/llama-models/blob/main/models/llama4/LICENSE",
		"bigscience-openrail-m": "https://huggingface.co/spaces/bigscience/license",
		"openrail":              "https://www.licenses.ai/ai-licenses",
		"gemma":                 "https://ai.google.dev/gemma/terms",
	}

	if url, exists := licenseURLs[licenseID]; exists {
		return url
	}

	return ""
}

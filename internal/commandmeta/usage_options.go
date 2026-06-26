package commandmeta

import "strings"

func optionsForUsage(usage []string, options []Value) []Value {
	if len(usage) == 0 || len(options) == 0 {
		return nil
	}
	referenced := map[string]bool{}
	for _, line := range usage {
		for _, field := range strings.Fields(line) {
			if option := usageOptionName(field); option != "" {
				referenced[option] = true
			}
		}
	}
	out := make([]Value, 0, len(options))
	for index, option := range options {
		if referenced[option.Name] || optionAliasOfReferenced(options, index, referenced) {
			out = append(out, option)
		}
	}
	return out
}

func optionAliasOfReferenced(options []Value, index int, referenced map[string]bool) bool {
	option := options[index].Name
	if !strings.HasPrefix(option, "-") || strings.HasPrefix(option, "--") {
		return false
	}
	if index > 0 && referenced[options[index-1].Name] {
		return true
	}
	return index+1 < len(options) && referenced[options[index+1].Name]
}

func usageOptionName(field string) string {
	field = strings.Trim(field, "[],")
	if strings.HasPrefix(field, "--") {
		return field
	}
	if strings.HasPrefix(field, "-") && len(field) == 2 {
		return field
	}
	return ""
}

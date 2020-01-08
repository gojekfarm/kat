package util

import "regexp"

func Filter(list []string, regex string, include bool) ([]string, error) {
	var filteredList []string
	for _, value := range list {
		matched, err := regexp.Match(regex, []byte(value))
		if err != nil {
			return nil, err
		}

		if (include && matched) || (!include && !matched) {
			filteredList = append(filteredList, value)
		}
	}
	return filteredList, nil
}
package util

import "regexp"

var reMinecraftCodes = regexp.MustCompile(`§.`)

func CleanMinecraftTags(input string) string {
	return reMinecraftCodes.ReplaceAllString(input, "")
}

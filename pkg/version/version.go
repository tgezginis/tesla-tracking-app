package version

const (
	Major = 1
	Minor = 0
	Patch = 0
	Build = ""
)

func String() string {
	return VersionString(Major, Minor, Patch, Build)
}

func VersionString(major, minor, patch int, build string) string {
	version := ""
	version += "v"
	version += IntToString(major)
	version += "."
	version += IntToString(minor)
	version += "."
	version += IntToString(patch)
	if build != "" {
		version += "+"
		version += build
	}
	return version
}

func IntToString(num int) string {
	return string(rune('0' + num))
} 
package gitenv

import "sort"

var repositoryDirectiveNames = map[string]struct{}{
	"GIT_ALTERNATE_OBJECT_DIRECTORIES": {},
	"GIT_COMMON_DIR":                   {},
	"GIT_DIR":                          {},
	"GIT_INDEX_FILE":                   {},
	"GIT_NAMESPACE":                    {},
	"GIT_OBJECT_DIRECTORY":             {},
	"GIT_WORK_TREE":                    {},
}

// RepositoryDirectiveNames returns Git environment variables that redirect
// repository discovery, working-tree selection, index files, or object storage.
func RepositoryDirectiveNames() []string {
	names := make([]string, 0, len(repositoryDirectiveNames))
	for name := range repositoryDirectiveNames {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func IsRepositoryDirective(name string) bool {
	_, ok := repositoryDirectiveNames[name]
	return ok
}

func RemoveRepositoryDirectives(env map[string]string) {
	for name := range repositoryDirectiveNames {
		delete(env, name)
	}
}

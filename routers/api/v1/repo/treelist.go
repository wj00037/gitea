// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"net/http"

	"code.gitea.io/gitea/modules/base"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/git"
)

// TreeList get all files' entries of a repository
func TreeList(ctx *context.APIContext) {
	tree, err := ctx.Repo.Commit.SubTree("/")
	if err != nil {
		ctx.ServerError("Repo.Commit.SubTree", err)
		return
	}

	entries, err := tree.ListEntriesRecursiveFast()
	if err != nil {
		ctx.ServerError("ListEntriesRecursiveFast", err)
		return
	}
	entries.CustomSort(base.NaturalSortLess)

	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !isExcludedEntry(entry) {
			files = append(files, entry.Name())
		}
	}
	ctx.JSON(http.StatusOK, files)
}

func isExcludedEntry(entry *git.TreeEntry) bool {
	if entry.IsDir() {
		return true
	}

	if entry.IsSubModule() {
		return true
	}

	return false
}

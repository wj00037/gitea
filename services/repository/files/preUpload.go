package files

import (
	"context"

	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/structs"
)

// FetchFileModesOptions holds the repository files info options
type FetchFileModesOptions struct {
	Files []*ChangeRepoFile
}

type PreUploadFile struct {
	SHA      string
	Size     int64
	TreePath string
}

type PreUploadFileInf struct {
	Branch string
	Files  []PreUploadFile
}

// FetchFileType determines the upload mode (LFS or regular) for files before uploading.
func FetchFileType(ctx context.Context, repo *repo_model.Repository, opts *PreUploadFileInf) (*structs.PreUploadFilesResponse, error) {

	// Create a temporary repository based on the context and repo provided
	tempUploadRepo, err := NewTemporaryUploadRepository(ctx, repo)
	if err != nil {
		return nil, err
	}

	// Clear temporary repository
	defer tempUploadRepo.Close()

	// Attempt to clone the repository for the given branch
	if err := tempUploadRepo.FullyClone(opts.Branch); err != nil {
		return nil, err
	}

	return LFSOrRegularFile(tempUploadRepo, opts)
}

// LFSOrRegularFile checks if the provided files should be handled as LFS or regular files.
func LFSOrRegularFile(tempRepo *TemporaryUploadRepository, infos *PreUploadFileInf) (*structs.PreUploadFilesResponse, error) {
	// Collect filenames from the infos parameter
	filenames := make([]string, len(infos.Files))
	for i, file := range infos.Files {
		filenames[i] = file.TreePath
	}

	// Check git attributes for all filenames
	attributes, err := tempRepo.gitRepo.CheckAttribute(git.CheckAttributeOpts{
		Attributes: []string{"filter"},
		Filenames:  filenames,
		CachedOnly: false,
	})
	if err != nil {
		return nil, err
	}

	return GetPreUploadFileResponse(attributes, infos.Files)
}

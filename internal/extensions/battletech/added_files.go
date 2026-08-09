package battletech

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

func adoptGeneratedFiles(ctx context.Context, input sdk.EventHandlerInput) (sdk.EventHandlerResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.EventHandlerResult{}, err
	}
	var result sdk.EventHandlerResult
	for _, file := range input.AddedFiles {
		if len(file.Candidates) != 1 {
			continue
		}
		candidate := file.Candidates[0]
		if strings.TrimSpace(candidate.ModType) == "" || strings.TrimSpace(candidate.StagingPath) == "" {
			continue
		}
		rel, err := cleanAddedFileRelative(file.TargetRelative)
		if err != nil {
			return sdk.EventHandlerResult{}, err
		}
		targetPath := filepath.Join(candidate.StagingPath, filepath.FromSlash(rel))
		same, err := sameFile(file.FilePath, targetPath)
		if err != nil {
			return sdk.EventHandlerResult{}, err
		}
		if !same {
			if err := copyRegularFile(file.FilePath, targetPath); err != nil {
				return sdk.EventHandlerResult{}, err
			}
		}
		if err := os.Remove(file.FilePath); err != nil && !errors.Is(err, os.ErrNotExist) && !same {
			return sdk.EventHandlerResult{}, err
		}
		result.AdoptedFiles = append(result.AdoptedFiles, sdk.AdoptedFile{
			InstalledModID:  candidate.InstalledModID,
			StagingRelative: rel,
			TargetRootID:    file.TargetRootID,
			TargetRelative:  rel,
		})
	}
	if len(result.AdoptedFiles) > 0 {
		result.Messages = append(result.Messages, "Adopted "+strconv.Itoa(len(result.AdoptedFiles))+" generated BattleTech file"+plural(len(result.AdoptedFiles)))
	}
	return result, nil
}

func cleanAddedFileRelative(value string) (string, error) {
	value = filepath.Clean(filepath.FromSlash(strings.TrimSpace(value)))
	if value == "." || value == "" || filepath.IsAbs(value) || strings.HasPrefix(filepath.ToSlash(value), "../") {
		return "", errors.New("unsafe BattleTech generated file path")
	}
	return filepath.ToSlash(value), nil
}

func sameFile(left, right string) (bool, error) {
	leftInfo, err := os.Stat(left)
	if err != nil {
		return false, err
	}
	rightInfo, err := os.Stat(right)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	return os.SameFile(leftInfo, rightInfo), nil
}

func copyRegularFile(source, target string) error {
	sourceFile, err := os.Open(source)
	if err != nil {
		return err
	}
	defer sourceFile.Close()
	info, err := sourceFile.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return errors.New("BattleTech generated file is not a regular file")
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		return err
	}
	targetFile, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return err
	}
	if _, err := io.Copy(targetFile, sourceFile); err != nil {
		_ = targetFile.Close()
		return err
	}
	return targetFile.Close()
}

func plural(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}

// Copyright 2022 individual contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// <https://www.apache.org/licenses/LICENSE-2.0>
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing
// permissions and limitations under the License.

package atomicpaths

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var randSource = rand.Reader

var randBitsEncoding = base32.HexEncoding.WithPadding(base32.NoPadding)

func makeTempName(originalName string) (string, error) {
	var rnd [5]byte
	if _, err := io.ReadFull(randSource, rnd[:]); err != nil {
		return "", fmt.Errorf("read rand: %w", err)
	}
	randBits := strings.ToLower(randBitsEncoding.EncodeToString(rnd[:]))

	name := ".#" + originalName + "-" + randBits + ".tmp"
	return name, nil
}

func makeTempPath(origPath string) (string, error) {
	origPath = filepath.Clean(origPath)

	// Specifying an empty path or the root directory is invalid.
	if len(origPath) == 0 || origPath[len(origPath)-1] == filepath.Separator {
		return "", fmt.Errorf("%q: %w", origPath, os.ErrInvalid)
	}

	name, err := makeTempName(filepath.Base(origPath))
	if err != nil {
		return "", err
	}

	return filepath.Join(filepath.Dir(origPath), name), nil
}

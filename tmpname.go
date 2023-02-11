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
	"os"
	"path/filepath"
	"strings"
)

var randBitsEncoding = base32.HexEncoding.WithPadding(base32.NoPadding)

func makeTempName(origPath string) (string, error) {
	origPath = filepath.Clean(origPath)

	// Specifying an empty path or the root directory is invalid.
	if len(origPath) == 0 || origPath[len(origPath)-1] == filepath.Separator {
		return "", fmt.Errorf("%q: %w", origPath, os.ErrInvalid)
	}

	var rnd [5]byte
	if _, err := rand.Read(rnd[:]); err != nil {
		return "", fmt.Errorf("read rand: %w", err)
	}
	randBits := strings.ToLower(randBitsEncoding.EncodeToString(rnd[:]))

	name := ".#" + filepath.Base(origPath) + "-" + randBits + ".tmp"
	return filepath.Join(filepath.Dir(origPath), name), nil
}

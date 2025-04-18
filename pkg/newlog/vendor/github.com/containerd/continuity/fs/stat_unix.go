//go:build linux || openbsd || dragonfly || solaris

/*
   Copyright The containerd Authors.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package fs

import (
	"fmt"
	"io/fs"
	"syscall"
	"time"
)

func Atime(st fs.FileInfo) (time.Time, error) {
	stSys, ok := st.Sys().(*syscall.Stat_t)
	if !ok {
		return time.Time{}, fmt.Errorf("expected st.Sys() to be *syscall.Stat_t, got %T", st.Sys())
	}
	return StatATimeAsTime(stSys), nil
}

func Ctime(st fs.FileInfo) (time.Time, error) {
	stSys, ok := st.Sys().(*syscall.Stat_t)
	if !ok {
		return time.Time{}, fmt.Errorf("expected st.Sys() to be *syscall.Stat_t, got %T", st.Sys())
	}
	return time.Unix(stSys.Atim.Unix()), nil
}

func Mtime(st fs.FileInfo) (time.Time, error) {
	stSys, ok := st.Sys().(*syscall.Stat_t)
	if !ok {
		return time.Time{}, fmt.Errorf("expected st.Sys() to be *syscall.Stat_t, got %T", st.Sys())
	}
	return time.Unix(stSys.Mtim.Unix()), nil
}

// StatAtime returns the Atim
func StatAtime(st *syscall.Stat_t) syscall.Timespec {
	return st.Atim
}

// StatCtime returns the Ctim
func StatCtime(st *syscall.Stat_t) syscall.Timespec {
	return st.Ctim
}

// StatMtime returns the Mtim
func StatMtime(st *syscall.Stat_t) syscall.Timespec {
	return st.Mtim
}

// StatATimeAsTime returns st.Atim as a time.Time
func StatATimeAsTime(st *syscall.Stat_t) time.Time {
	return time.Unix(st.Atim.Unix())
}

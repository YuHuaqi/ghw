// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.
//

package ghw

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func cachesForNode(nodeID int) ([]*MemoryCache, error) {
	// The /sys/devices/node/nodeX directory contains a subdirectory called
	// 'cpuX' for each logical processor assigned to the node. Each of those
	// subdirectories containers a 'cache' subdirectory which contains a number
	// of subdirectories beginning with 'index' and ending in the cache's
	// internal 0-based identifier. Those subdirectories contain a number of
	// files, including 'shared_cpu_list', 'size', and 'type' which we use to
	// determine cache characteristics.
	path := filepath.Join(
		pathSysDevicesSystemNode(),
		fmt.Sprintf("node%d", nodeID),
	)
	caches := make(map[string]*MemoryCache, 0)

	files, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		filename := file.Name()
		if !strings.HasPrefix(filename, "cpu") {
			continue
		}
		if filename == "cpumap" || filename == "cpulist" {
			// There are two files in the node directory that start with 'cpu'
			// but are not subdirectories ('cpulist' and 'cpumap'). Ignore
			// these files.
			continue
		}
		// Grab the logical processor ID by cutting the integer from the
		// /sys/devices/system/node/nodeX/cpuX filename
		cpuPath := filepath.Join(path, filename)
		lpID, _ := strconv.Atoi(filename[3:])

		// Inspect the caches for each logical processor. There will be a
		// /sys/devices/system/node/nodeX/cpuX/cache directory containing a
		// number of directories beginning with the prefix "index" followed by
		// a number. The number indicates the level of the cache, which
		// indicates the "distance" from the processor. Each of these
		// directories contains information about the size of that level of
		// cache and the processors mapped to it.
		cachePath := filepath.Join(cpuPath, "cache")
		cacheDirFiles, err := ioutil.ReadDir(cachePath)
		if err != nil {
			return nil, err
		}
		for _, cacheDirFile := range cacheDirFiles {
			cacheDirFileName := cacheDirFile.Name()
			if !strings.HasPrefix(cacheDirFileName, "index") {
				continue
			}

			typePath := filepath.Join(cachePath, cacheDirFileName, "type")
			cacheTypeContents, err := ioutil.ReadFile(typePath)
			if err != nil {
				continue
			}
			var cacheType MemoryCacheType
			switch string(cacheTypeContents[:len(cacheTypeContents)-1]) {
			case "Data":
				cacheType = DATA
			case "Instruction":
				cacheType = INSTRUCTION
			default:
				cacheType = UNIFIED
			}
			level := memoryCacheLevel(nodeID, lpID)
			size := memoryCacheSize(nodeID, lpID, level)

			scpuPath := filepath.Join(
				cachePath,
				cacheDirFileName,
				"shared_cpu_map",
			)
			sharedCpuMap, err := ioutil.ReadFile(scpuPath)
			if err != nil {
				continue
			}
			// The cache information is repeated for each node, so here, we
			// just ensure that we only have a one MemoryCache object for each
			// unique combination of level, type and processor map
			cacheKey := fmt.Sprintf("%d-%d-%s", level, cacheType, sharedCpuMap[:len(sharedCpuMap)-1])
			cache, exists := caches[cacheKey]
			if !exists {
				cache = &MemoryCache{
					Level:             uint8(level),
					Type:              cacheType,
					SizeBytes:         uint64(size) * uint64(KB),
					LogicalProcessors: make([]uint32, 0),
				}
				caches[cacheKey] = cache
			}
			cache.LogicalProcessors = append(
				cache.LogicalProcessors,
				uint32(lpID),
			)
		}
	}

	cacheVals := make([]*MemoryCache, len(caches))
	x := 0
	for _, c := range caches {
		// ensure the cache's processor set is sorted by logical process ID
		sort.Sort(SortByLogicalProcessorId(c.LogicalProcessors))
		cacheVals[x] = c
		x++
	}

	return cacheVals, nil
}

func pathNodeCPU(nodeID int, lpID int) string {
	return filepath.Join(
		pathSysDevicesSystemNode(),
		fmt.Sprintf("node%d", nodeID),
		fmt.Sprintf("cpu%d", lpID),
	)
}

func pathNodeCPUCache(nodeID int, lpID int) string {
	return filepath.Join(
		pathNodeCPU(nodeID, lpID),
		"cache",
	)
}

func pathNodeCPUCacheLevel(nodeID int, lpID int, cacheLevel int) string {
	return filepath.Join(
		pathNodeCPUCache(nodeID, lpID),
		fmt.Sprintf("index%d", cacheLevel),
	)
}

func memoryCacheSize(nodeID int, lpID int, cacheLevel int) int {
	sizePath := filepath.Join(
		pathNodeCPUCacheLevel(nodeID, lpID, cacheLevel),
		"size",
	)
	sizeContents, err := ioutil.ReadFile(sizePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to read %s: %s", sizePath, err)
		return -1
	}
	// size comes as XK\n, so we trim off the K and the newline.
	size, err := strconv.Atoi(string(sizeContents[:len(sizeContents)-2]))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to parse int from %s", sizeContents)
		return -1
	}
	return size
}

func memoryCacheLevel(nodeID int, lpID int) int {
	levelPath := filepath.Join(
		pathNodeCPUCache(nodeID, lpID),
		"level",
	)
	levelContents, err := ioutil.ReadFile(levelPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to read %s: %s", levelPath, err)
		return -1
	}
	// levelContents is now a []byte with the last byte being a newline
	// character. Trim that off and convert the contents to an integer.
	level, err := strconv.Atoi(string(levelContents[:len(levelContents)-1]))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to parse int from %s", levelContents)
		return -1
	}
	return level
}

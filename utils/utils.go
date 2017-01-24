/*
Package utils contains all the helper functions for navigator.
*/
package utils

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"runtime/debug"

	"math/rand"
	"time"

	"sort"
	"strings"

	"runtime"

	"net"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/dgryski/go-farm"
)

// this is the L1 cached value for the hostname
var (
	logTag         = "Util"
	cachedHostname = ""
	getHostLocker  sync.Mutex
)

// GetMD5Hash will generate a md5 hash key from the string
func GetMD5Hash(text string) string {
	hasher := md5.New()
	_, err := hasher.Write([]byte(text))
	if err != nil {
		log.Warnf(logTag, "Error GetMD5Hash %s, %s", text, err)
	}
	return hex.EncodeToString(hasher.Sum(nil))
}

// GetJSONStr will return an mashaled json string from struct, if failed, return empty string
func GetJSONStr(t interface{}) string {
	item, err := json.Marshal(t)
	if err == nil {
		return string(item)
	}
	log.Warn(logTag, fmt.Sprintf("Failed to marshal json object. %s", err))
	return ""
}

// Difference will find the difference between two slices,
// and return items in slice1 not in slice2,
// and return items in slice2 not in slice1
func Difference(slice1 []string, slice2 []string) ([]string, []string) {
	var diff1 []string
	var diff2 []string

	// Loop two times, first to find slice1 strings not in slice2,
	// second loop to find slice2 strings not in slice1
	for i := 0; i < 2; i++ {
		for _, s1 := range slice1 {
			found := false
			for _, s2 := range slice2 {
				if s1 == s2 {
					found = true
					break
				}
			}
			// String not found. We add it to return slice
			if !found {
				if i == 0 {
					diff1 = append(diff1, s1)
				} else {
					diff2 = append(diff2, s1)
				}
			}
		}
		// Swap the slices, only if it was the first loop
		if i == 0 {
			slice1, slice2 = slice2, slice1
		}
	}

	return diff1, diff2
}

// ShuffleStrings takes a slice of strings and returns a new slice containing
// the same strings in a random order.
func ShuffleStrings(strings []string) []string {
	newStrings := make([]string, len(strings))
	newIndexes := rand.Perm(len(strings))

	for o, n := range newIndexes {
		newStrings[n] = strings[o]
	}

	return newStrings
}

// ShuffleStringsInPlace uses the Fisher–Yates shuffle to randomize the strings
// in place.
func ShuffleStringsInPlace(strings []string) {
	for i := range strings {
		j := rand.Intn(i + 1)
		strings[i], strings[j] = strings[j], strings[i]
	}
}

// TakeNode takes an element from nodes at the given index, or at a random index if
// index < 0. Mutates nodes.
func TakeNode(nodes *[]string, index int) string {
	if len(*nodes) == 0 {
		return ""
	}

	var i int
	if index >= 0 {
		if index >= len(*nodes) {
			return ""
		}
		i = index
	} else {
		i = rand.Intn(len(*nodes))
	}

	node := (*nodes)[i]

	*nodes = append((*nodes)[:i], (*nodes)[i+1:]...)

	return node
}

// SelectInt takes an option and a default value and returns the default value if
// the option is equal to zero, and the option otherwise.
func SelectInt(opt, def int) int {
	if opt == 0 {
		return def
	}
	return opt
}

// SelectFloat takes an option and a default value and returns the default value if
// the option is equal to zero, and the option otherwise.
func SelectFloat(opt, def float64) float64 {
	if opt == 0 {
		return def
	}
	return opt
}

// SelectDuration takes an option and a default value and returns the default value if
// the option is equal to zero, and the option otherwise.
func SelectDuration(opt, def time.Duration) time.Duration {
	if opt == time.Duration(0) {
		return def
	}
	return opt
}

// SelectTime takes an option and a default value and returns the default value if
// the option is equal to zero, and the option otherwise.
func SelectTime(opt, def time.Time) time.Time {
	if opt.IsZero() {
		return def
	}
	return opt
}

// SelectString takes an option and a default value and returns the default value if
// the option is equal to zero, and the option otherwise.
func SelectString(opt, def string) string {
	if opt == "" {
		return def
	}
	return opt
}

// Min returns min(a,b)
func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// GetHostname returns a unique identifier for the current machine
// (currently implemented as the ip of the machine)
var GetHostname = func() string {
	getHostLocker.Lock()
	defer getHostLocker.Unlock()
	if cachedHostname != "" {
		return cachedHostname
	}

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		log.Warnf(logTag, "getSelfID failed to retrieve any network interfaces on the current machine, %s", err)
		return ""
	}
	for _, addr := range addrs {
		// skip the loopback interface and any non IPv4 interfaces
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				cachedHostname = ipnet.IP.String()
				return cachedHostname
			}
		}
	}
	log.Errorf(logTag, "getSelfID failed to find any usable network interfaces on the current machine", err)
	return ""
}

// GetCheckSumFromNodes will Get the checkSum from nodes
func GetCheckSumFromNodes(nodes []string) uint32 {
	sort.Strings(nodes)
	bytes := []byte(strings.Join(nodes, ";"))
	return farm.Fingerprint32(bytes)
}

// GetErrStr will get a string representation of error
func GetErrStr(err error) string {
	s := err.Error()
	return fmt.Sprintf("version: %s\ntypes: %T / %T\nstring value via err.Error(): %q\n", runtime.Version(), err, s, s)
}

// DoPanicRecovery is the common panic recover pattern for go routing
// All the go routing normal failure should return error, instead of panic.
func DoPanicRecovery(name string) {
	if r := recover(); r != nil {
		log.Errorf(logTag, "%s failed with error %s %s", name, r, string(debug.Stack()))
	}
}

// StrSliceContains will return whether slice contains the value
func StrSliceContains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

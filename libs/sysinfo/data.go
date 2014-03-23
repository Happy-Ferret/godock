/*
Package sysinfo is a library that get some linux system information.
*/
package sysinfo

import (
	"bufio"
	"errors"
	"os"
	"strconv"
	"strings"
)

//
//------------------------------------------------------[ DISK ACTIVITY DATA ]--

const KernelDiskStats = "/proc/diskstats"

const ( // Position of our data in the kernel file.
	diskName = 2
	diskIn   = 5
	diskOut  = 9
)

// Get activity information for configured disks.
//
// Using Linux iostat : http://www.kernel.org/doc/Documentation/iostats.txt
//
func GetDiskActivity() ([]value, error) {
	file, e := os.Open(KernelDiskStats)
	if e != nil {
		return nil, errors.New("Your kernel doesn't support diskstat. (2.5.70 or above required)")
	}
	defer file.Close()

	var values []value

	r := bufio.NewReader(file)
	line, err := r.ReadString('\n')
	for ; err == nil; line, err = r.ReadString('\n') {
		data := strings.Fields(line) // Useful data is only separated by a blank space.

		if len(data) < diskOut || len(data[diskName]) == 0 {
			return nil, errors.New("DiskActivity: failed parsing")
		}

		last := data[diskName][len(data[diskName])-1]

		if last > '9' { // Drop interfaces where last char is a number.
			in, _ := strconv.ParseUint(data[diskIn], 10, 64)
			out, _ := strconv.ParseUint(data[diskOut], 10, 64)

			values = append(values, value{
				Field: data[diskName],
				In:    in,
				Out:   out})
		}

		// line, err = r.ReadString('\n')
	}

	return values, nil
}

//
//-------------------------------------------------------[ NET ACTIVITY DATA ]--

const KernelNetStats = "/proc/net/dev"

const ( // Position of our data in the kernel file.
	netName = 0
	netIn   = 1
	netOut  = 9
)

func GetNetActivity() ([]value, error) {
	file, e := os.Open(KernelNetStats)
	if e != nil {
		return nil, e
	}
	defer file.Close()
	var values []value

	r := bufio.NewReader(file)

	r.ReadString('\n') // Drop first two lines with fields names.
	r.ReadString('\n')
	line, err := r.ReadString('\n')
	for ; err == nil; line, err = r.ReadString('\n') {
		data := strings.Fields(line) // Useful data is only separated by spaces.

		if len(data) < netOut || len(data[netName]) == 0 {
			return nil, errors.New("NetActivity: failed parsing")
		}

		if data[0] != "lo:" { // Drop loopback interface.
			in, _ := strconv.ParseUint(data[netIn], 10, 64)
			out, _ := strconv.ParseUint(data[netOut], 10, 64)

			values = append(values, value{
				Field: data[netName][:len(data[netName])-1], // Remove ":" at the end of interface name.
				In:    in,
				Out:   out})
		}
	}
	return values, nil
}

//
//-------------------------------------------------------------[ DELTA STATS ]--

type stat struct {
	rateReadNow  uint64
	rateWriteNow uint64

	rateReadMax  uint64
	rateWriteMax uint64

	blocksRead  uint64
	blocksWrite uint64

	bInitialized  bool // true after the 2nd data pull.
	acquisitionOK bool // true if data was found this pull.
}

func (st *stat) Set(in, out, interval uint64) {
	st.acquisitionOK = true

	if st.bInitialized { // Drop first pull. Values are stupidly high: total since boot time.
		st.rateReadNow = (in - st.blocksRead) / interval
		st.rateWriteNow = (out - st.blocksWrite) / interval
	}

	// Save our new values.
	st.blocksRead = in
	st.blocksWrite = out
	st.bInitialized = true
}

func (st *stat) Current() (in, out float64, ok bool) {
	in = currentRate(st.rateReadNow, &st.rateReadMax)
	out = currentRate(st.rateWriteNow, &st.rateWriteMax)
	return in, out, st.acquisitionOK
}

func currentRate(speed uint64, max *uint64) float64 {
	if speed > *max {
		*max = speed
	}
	if *max > 0 {
		return float64(speed) / float64(*max)
	}
	return 0
}

type value struct {
	Field string
	In    uint64
	Out   uint64
}

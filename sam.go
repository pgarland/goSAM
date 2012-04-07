// Copyright (C) 2012 Phillip Garland <pgarland@gmail.com>

// This program is free software: you can redistribute it and/or
// modify it under the terms of the GNU Lesser General Public License
// as published by the Free Software Foundation, either version 3 of
// the License, or (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.

// You should have received a copy of the GNU Lesser General Public
// License along with this program.  If not, see
// <http://www.gnu.org/licenses/>.

package goSAM

import (
	"fmt"
	"os"
	"bufio"
	"strings"
	"strconv"
	"container/list"
	"regexp"
)

type HeaderLine struct {
	Version string // VN | /^[0-9]+\.[0-9]+$/ | required
	SortOrder string // SO | unknown, unsorted, queryname, coordinate | optional
}

func validateHeader(hl *HeaderLine) (bool, error) {
	m, _ := regexp.Match("^[0-9]+.[0-9]+$", []byte(hl.Version))
	if !m {
		return m, SAMerror{"Invalid version in SAM Header"}
	} 
	return m, nil

}

var hlParseMap = map[string]func(string, *HeaderLine) {
	"VN": func(val string, hl *HeaderLine) {hl.Version = val},
	"SO": func(val string, hl *HeaderLine) {hl.SortOrder = val},
}

func parseHeader(line string) *HeaderLine {
	tvs := strings.Split(line, "\t")
	hl := HeaderLine{}
	for _,tv := range tvs[1:] {
		tva := strings.Split(tv,":")
		tag := tva[0]
		val := tva[1]
		parseFunc := hlParseMap[tag]
		if parseFunc != nil {
			parseFunc(val, &hl)
		} // FIXME: catch and collect non-std tags?
	}
	return &hl
}

// Order of SQ lines defines the alignment sorting order
type RefSeqDict struct {
	Name string // SN | [!-)+-<>-~][!-~]*  | required | unique
	Length uint32 // LN | Range: [1, 2^29 -1] | required
	AssemblyID string // AS | optional
	MD5 string // M5 | optional
	Species string // SP | optional
	URI string // || UR | optional | use URL type?
}

func validateRefSeqDict(rsd *RefSeqDict) (bool, error) {
	m , _ := regexp.Match("[!-)+-<>-~][!-~]*", []byte(rsd.Name))
	if !m {
		return false, SAMerror{"Invalid reference sequence name"}
	}

	return ((rsd.Length >= 1) && (rsd.Length <= 0x1FFFFFFF)), nil
}

func parseRefSeqDict(line string) *RefSeqDict {
	tvs := strings.Split(line, "\t")
	rsd := RefSeqDict{}
	for _,tv := range tvs[1:] {
		tva := strings.Split(tv,":")
		switch tag := tva[0]; tag {
		case "SN":
			rsd.Name = tva[1]
		case "LN":
			v, _ := strconv.Atoi(tva[1])
			rsd.Length = uint32(v)
		case "AS":
			rsd.AssemblyID = tva[1]
		case "M5":
			rsd.MD5 = tva[1]
		case "SP":
			rsd.Species = tva[1]
		case "UR":
			rsd.URI = tva[1]
		}
	}
	return &rsd
}

type ReadGroup struct {
	ID string // ID | unique | required
	SeqCenter string // CN | optional 
	Description string // DS | optional
	Date string // DT | optional
	FlowOrder string // FO | /\*|[ACMGRSVTWYHKDBN]+/ | optional
	KeySeq string // KS | optional
	Lib string // LB | optional
	Programs string // PG | optional
	PMIS string // PI | optional | predicted median insert size
	Platform string // PL | CAPILLARY LS454 ILLUMINA SOLID HELICOS IONTORRENT PACBIO | optional
	Unit string // PU | Unique | optional
	Sample string // SM | optional
}

// The usefulness of checking platforms seems dubious to me. What
// happens as new platforms come into use. I may skip checking this if
// it causes problems.
var validPlatforms = map[string]bool{
	"CAPILLARY": true,
	"LS454": true,
	"ILLUMINA": true,
	"SOLID": true, 
	"HELICOS": true, 
	"IONTORRENT": true,
	"PACBIO": true,
}

// FIXME: make sure ID is unique
func validateReadGroup (rg *ReadGroup) (bool, error) {
	m := true
	// FlowOrder is optional, so we have to check it's existence
	// first, though I guess I could just include the empty string as
	// an alternative in the match.
	if rg.FlowOrder != "" {
		m, _ = regexp.Match("*|[ACMGRSVTWYHKDBN]+",[]byte(rg.FlowOrder))
		if !m {
			return false, SAMerror{"Invalid flow order in read group"}
		}
	}
	if rg.Platform != "" {
		m = validPlatforms[rg.Platform]
		if !m {return false, SAMerror{"Invalid platform in read group"}}
	}
	return true, nil
}

var rgParseMap = map[string]func(string, *ReadGroup) {
	"ID": func(s string, rg *ReadGroup) {rg.ID = s},
	"CN": func(s string, rg *ReadGroup) {rg.SeqCenter = s},
	"DS": func(s string, rg *ReadGroup) {rg.Description = s},
	"DT": func(s string, rg *ReadGroup) {rg.Date = s},
	"FO": func(s string, rg *ReadGroup) {rg.FlowOrder = s},
	"KS": func(s string, rg *ReadGroup) {rg.KeySeq = s},
	"LB": func(s string, rg *ReadGroup) {rg.Lib = s},
	"PG": func(s string, rg *ReadGroup) {rg.Programs = s},
	"PI": func(s string, rg *ReadGroup) {rg.PMIS = s},
	"PL": func(s string, rg *ReadGroup) {rg.Platform = s},
	"PU": func(s string, rg *ReadGroup) {rg.Unit = s},
	"SM": func(s string, rg *ReadGroup) {rg.Sample = s},
}

func parseReadGroup(line string) *ReadGroup {
	tvs := strings.Split(line, "\t")
	rg := ReadGroup{}
	for _,tv := range tvs[1:] {
		tva := strings.Split(tv,":")
		tag := tva[0]
		val := tva[1]
		parseFunc := rgParseMap[tag]
		if parseFunc != nil {
			parseFunc(val, &rg)
		} // FIXME: catch and collect non-std tags?
	}
	return &rg
}

type Program struct {
	ID string // ID | unique | required
	Name string // PN | optional
	CmdLine string // CL | optional
	PrevID string // PP | must match another PG line ID | optional
}

func validateProgram(prog *Program) (bool, error) {
	if prog.ID == "" {return false, SAMerror{"Program ID is required"}}
	return true, nil
}

var programParseMap = map[string]func(string, *Program) {
	"ID": func(s string, prog *Program) {prog.ID = s},
	"PN": func(s string, prog *Program) {prog.Name = s},
	"CL": func(s string, prog *Program) {prog.CmdLine = s},
	"PP": func(s string, prog *Program) {prog.PrevID = s},
}	

func parseProgram(line string) *Program {
	tvs := strings.Split(line, "\t")
	prog := Program{}
	for _,tv := range tvs[1:] {
		tva := strings.Split(tv,":")
		tag := tva[0]
		val := tva[1]
		parseFunc := programParseMap[tag]
		if parseFunc != nil {
			parseFunc(val, &prog)
		} // FIXME: catch and collect non-std tags?
	}
	return &prog
}

type Alignment struct {
	Qname string // required | [!-?A-~]{1-255} | query template name
	Flag uint16 // required | [0-2^16 - 1] | bitwise flag
	RefName string // required | \*|[!-()+-<>-~][!-~]*
	Pos uint32 // required | [0-2^29-1]
	Mapq uint8 // required | [0-2^8-1]
	Cigar string // required | \*|([0-9]+[MIDNSHPX=])+
	NextRef string // required | \*|=|[!-()+-<>-~][!-~]*
	NextPos uint32 // required | [0-2^29-1]
	TemplateLen int32 // required | [-2^29+1 - 2^29-1]
	Seq string // required | \*|[A-Za-z=.]+
	Qual string // required ASCII Phred score+33
}

// FIXME: These regexp patterns should be compiled, since they'll be
// used over and over
func validateAlignment(a *Alignment) (bool, error){
	if m, _ := regexp.Match("*|[!-?A-~]+", []byte(a.Qname)); !m {
		return false, SAMerror{"Invalid qname in alignment"}
	}
	if (a.Flag < 0 || a.Flag > 0xFFFF) {
		return false, SAMerror{"Invalid flag in alignment"}
	}
	if m, _ := regexp.Match("*|[!-()+-<>-~][!-~]*", []byte(a.RefName)); !m {
		return false, SAMerror{"Invalid reference sequence name in alignment"}
	}
	if a.Pos < 0 || a.Pos > 0x1FFFFFFF {
		return false, SAMerror{"Alignment mapping position out of valid range"}
	}
	if a.Mapq < 0 || a.Mapq > 0xFF {
		return false, SAMerror{"Alignment mapping quality out of valid range"}
	}
	if m, _ := regexp.Match("*|([0-9]+[MIDNSHPX=])+", []byte(a.Cigar)); !m {	
		return false, SAMerror{"Invalid CIGAR string in alignment"}
	}
	if m, _ := regexp.Match("*|=|[!-()+-<>-~][!-~]*", []byte(a.NextRef)); !m {
		return false, SAMerror{"Invalid next reference name in alignment"}
	}
	if a.NextPos < 0 || a.NextPos > 0x1FFFFFFF {
		return false, SAMerror{"Alignment mapping position out of valid range"}
	}
	if a.TemplateLen < -0x1FFFFFFF || a.TemplateLen > 0x1FFFFFFF {
		return false, SAMerror{"Invalid template length"}
	}
	if m, _ := regexp.Match("*|[A-Za-z=.]+",[]byte(a.Seq)); !m {
		return false, SAMerror{"Invalid sequence in alignment"}
	}
	if m, _ := regexp.Match("*|[!-~]+",[]byte(a.Qual)); !m {
		return false, SAMerror{"Invalie Phred quality in alignment"}
	}	
	return true, nil
}
func parseAlignment(line string) *Alignment {
	fields := strings.Split(line, "\t")

	alignment := Alignment{}
	alignment.Qname = fields[0]

	flagVal, _ := strconv.Atoi(fields[1])
	alignment.Flag = uint16(flagVal)

	alignment.RefName = fields[2]

	posVal, _ := strconv.Atoi(fields[3])
	alignment.Pos = uint32(posVal)

	mapqVal,  _ := strconv.Atoi(fields[4])
	alignment.Mapq = uint8(mapqVal)

	alignment.Cigar = fields[5]
	alignment.NextRef = fields[6]

	nextPosVal, _ := strconv.Atoi(fields[7])
	alignment.NextPos = uint32(nextPosVal)

	templateLenVal, _ := strconv.Atoi(fields[8])
	alignment.TemplateLen = int32(templateLenVal)	

	alignment.Seq = fields[9]
	alignment.Qual = fields[10]

	return &alignment
}

type SAMerror struct {
	str string
}

func (e SAMerror) Error() string {
	return fmt.Sprintf("sam: %s", e.str)
}


func ReadSAMFile(fileName string) (*HeaderLine, *list.List, *list.List, *list.List, *list.List, error) {
	file, err := os.Open(fileName);
	if err != nil {
		fmt.Println(err)
        return nil, nil, nil, nil, nil, err
    }

	reader := bufio.NewReader(file)

	// These will be returned so they must be declared in this scope
	var header *HeaderLine
	var rsdl, rgl, progl, al = list.New(), list.New(), list.New(), list.New()

	// Maps to keep track of values that must be unique. Used for checking for duplicate values.
	var rsdNames, rgIDs, progIDs = map[string]bool{},  map[string]bool{}, map[string]bool{}

	// separating the cases into separate handler functions doesn't
	// seem to win much, so I'm leaving this as it is for now, though
	// it is longer than I'd like.
	for line, _, err := reader.ReadLine(); err == nil;  line, _, err = reader.ReadLine() {
		s := string(line)
		switch lineTag := s[1:3]; lineTag {
		case "HD": 		
			header = parseHeader(s)
			if valid, err := validateHeader(header); !valid {
					return header, nil, nil, nil, nil, err
			}
		case "SQ":
			rsd := parseRefSeqDict(s)
			if valid, err := validateRefSeqDict(rsd); !valid {
				return  header, nil, nil, nil, nil, err
			} else { 		
				if rsdNames[rsd.Name] { // Make sure name is unique
					return  header, rsdl, nil, nil, nil, SAMerror{"Reference sequence name is not unique"}
				} else { // Everything is OK
					rsdNames[rsd.Name] = true
					rsdl.PushBack(rsd)
				}
			}
		case "RG":
			rg := parseReadGroup(s)
			if valid, err := validateReadGroup(rg); !valid {
				return header, rsdl, rgl, nil, nil, err
			} else {
				if rgIDs[rg.ID] {
					return  header, rsdl, rgl, nil, nil, SAMerror{"Read group name is not unique"}
				} else {
					rgIDs[rg.ID] = true
					rgl.PushBack(rg)
				}
			}
		case "PG":
			prog := parseProgram(s)
			if valid, err := validateProgram(prog); !valid {
				return header, rsdl, rgl, progl, nil, err
			} else {
				if progIDs[prog.ID] {
					return header, rsdl, rgl, progl, nil, SAMerror{"Program ID is not unique"}
				} else {
					progIDs[prog.ID] = true
					progl.PushBack(prog)
				}
			}
		case "CO":
			// FIXME: It should be possible for the QNAME field of an
			// alignment to have "HD", "SQ", "RG", "PG", or "CO" as
			// characters 1 and 2, so making alignment the default
			// lone type is not right.
		default: 
			a := parseAlignment(s)
			if valid, err := validateAlignment(a); !valid {
				return header, rsdl, rgl, progl, al , err
			} else {
				al.PushBack(a)
			}
		}
	}

	file.Close()

	return header, rsdl, rgl, progl, al, err
}

func ReadNextAlignment() {
}

This is a library for reading SAM sequence alignment files, using the
Go programming language. It's in a pre-alpha state, and the interface
and implementation is subject to breaking change. It does read all the
required and optional data found in Header, Sequence Reference
Dictionary, Read Group, and Program lines, as decribed in the SAM
specification. It also reads all the required alignment data

It doesn't do any validation yet, either of lines, or of the entire
file.

Right now there is just one method:

func ReadSAMFile(fileName string) (*HeaderLine, *list.List, *list.List, *list.List, *list.List, error)

which returns a struct for the Header, as well as lists of structs for

The library is licensed according to the GNU Lesser GPL, Version 3. See COPYING.LESSER for details.
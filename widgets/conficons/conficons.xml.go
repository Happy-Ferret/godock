package conficons

import (
	"bytes"
	"compress/gzip"
	"io"
)

// conficonsXML returns raw, uncompressed file data.
func conficonsXML() []byte {
	gz, err := gzip.NewReader(bytes.NewBuffer([]byte{
		0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x00, 0xff, 0xbc, 0x55,
		0x4d, 0x6f, 0xdb, 0x30, 0x0c, 0xbd, 0xf7, 0x57, 0x68, 0xba, 0x0e, 0xae,
		0x97, 0xf5, 0xd2, 0x83, 0xed, 0x62, 0x28, 0xd0, 0x62, 0xc0, 0x30, 0x0c,
		0x6b, 0xb6, 0x1d, 0x0d, 0x45, 0x62, 0x62, 0x2d, 0x8a, 0xe4, 0x51, 0x74,
		0x9b, 0xfc, 0xfb, 0xc9, 0x1f, 0x49, 0x9c, 0xc4, 0x49, 0x93, 0x0d, 0xdb,
		0x25, 0x81, 0x44, 0xf2, 0x91, 0x7c, 0x7c, 0x94, 0x93, 0xbb, 0xe5, 0xc2,
		0xb0, 0x67, 0x40, 0xaf, 0x9d, 0x4d, 0xf9, 0xe8, 0xfa, 0x1d, 0x67, 0x60,
		0xa5, 0x53, 0xda, 0xce, 0x52, 0xfe, 0x6d, 0xfc, 0x10, 0xdd, 0xf2, 0xbb,
		0xec, 0x2a, 0x79, 0x13, 0x45, 0xec, 0x11, 0x2c, 0xa0, 0x20, 0x50, 0xec,
		0x45, 0x53, 0xc1, 0x66, 0x46, 0x28, 0x60, 0x37, 0xd7, 0xa3, 0xdb, 0xeb,
		0x1b, 0x16, 0x45, 0xc1, 0x49, 0x5b, 0x02, 0x9c, 0x0a, 0x09, 0xd9, 0x15,
		0x63, 0x09, 0xc2, 0xaf, 0x4a, 0x23, 0x78, 0x66, 0xf4, 0x24, 0xe5, 0x33,
		0x9a, 0xbf, 0xe5, 0xdb, 0x44, 0x37, 0x21, 0x51, 0xdc, 0xb8, 0xb9, 0xc9,
		0x4f, 0x90, 0xc4, 0xa4, 0x11, 0xde, 0xa7, 0xfc, 0x91, 0xe6, 0x9f, 0xb4,
		0xa7, 0x27, 0x72, 0x08, 0x9c, 0x69, 0x95, 0xf2, 0x85, 0x53, 0x60, 0x78,
		0xed, 0x1a, 0x9c, 0xa5, 0x33, 0xd5, 0xc2, 0xfa, 0xf6, 0x14, 0xce, 0x75,
		0x59, 0xed, 0x5d, 0x64, 0xc5, 0x02, 0xd8, 0xbd, 0xb3, 0xd3, 0xa6, 0x96,
		0xce, 0xde, 0xda, 0x18, 0xad, 0x4a, 0x08, 0x25, 0xc8, 0x42, 0xa0, 0x40,
		0x14, 0xab, 0x36, 0xf5, 0x20, 0xc2, 0x47, 0xe9, 0xec, 0x51, 0x84, 0x47,
		0x35, 0xff, 0xa2, 0x97, 0x93, 0x6a, 0x7a, 0x02, 0xe0, 0x73, 0xfd, 0x73,
		0x7e, 0x09, 0x49, 0xdc, 0xeb, 0x29, 0x89, 0x5b, 0x36, 0x86, 0x89, 0x79,
		0x92, 0xe8, 0x8c, 0x01, 0xf5, 0x43, 0x5b, 0xe5, 0x5e, 0x5a, 0x76, 0x5e,
		0xb4, 0x9a, 0x01, 0xad, 0xe9, 0x29, 0xd1, 0x95, 0x80, 0xb4, 0x62, 0x75,
		0x25, 0x29, 0x7f, 0xd6, 0x5e, 0x4f, 0x0c, 0xf0, 0x6c, 0x8c, 0x15, 0x24,
		0xf1, 0xda, 0x3a, 0xec, 0x2c, 0x85, 0xcd, 0xa7, 0x4e, 0x56, 0x9e, 0x67,
		0x0f, 0xc2, 0xf8, 0x57, 0xfd, 0x7d, 0x21, 0x42, 0x15, 0x79, 0xdd, 0x17,
		0xcf, 0x80, 0x64, 0x01, 0x2a, 0xd2, 0xf6, 0x20, 0x4a, 0x16, 0xda, 0xa8,
		0x0d, 0x19, 0x07, 0x3d, 0x8d, 0x11, 0xe0, 0xbb, 0x86, 0xae, 0x1b, 0x0a,
		0x27, 0xbe, 0x76, 0xbe, 0xb0, 0x9f, 0x57, 0x7a, 0x3a, 0x37, 0xa4, 0x10,
		0xfe, 0xd2, 0x90, 0x4e, 0xa1, 0xcd, 0xdf, 0x59, 0x29, 0x20, 0x2c, 0x0e,
		0xfa, 0x7c, 0xd3, 0xce, 0x20, 0xdf, 0xa7, 0x22, 0xa5, 0xd1, 0x72, 0x2e,
		0x2e, 0x8b, 0xc5, 0xca, 0x80, 0xcf, 0x8b, 0xb0, 0xa3, 0xe7, 0x77, 0xe6,
		0x41, 0xa0, 0x2c, 0xf2, 0x56, 0xa1, 0x3c, 0x7b, 0x3f, 0x18, 0xd3, 0x4c,
		0x98, 0x35, 0xbb, 0x6f, 0x85, 0x89, 0x9a, 0x63, 0x1d, 0x6a, 0xc2, 0xa4,
		0xc3, 0xae, 0xf7, 0xe6, 0x79, 0x4c, 0x00, 0x4f, 0x1b, 0xdf, 0x46, 0x05,
		0xdb, 0xd0, 0xb8, 0x97, 0x26, 0xde, 0x51, 0xd2, 0xbe, 0xb2, 0x4e, 0xab,
		0xeb, 0xbe, 0x6d, 0xa0, 0x41, 0x6f, 0x9b, 0xa9, 0xd7, 0x7c, 0xa7, 0xb2,
		0x21, 0xc0, 0x61, 0xd0, 0x7b, 0x30, 0xe6, 0x2b, 0xd8, 0x30, 0x08, 0xc0,
		0xee, 0x35, 0x68, 0x81, 0xc3, 0x7d, 0x03, 0x1b, 0x1f, 0x60, 0x08, 0x22,
		0xd4, 0x93, 0x8a, 0xc0, 0xef, 0x9b, 0xfa, 0xc6, 0x8e, 0xf4, 0xb2, 0xc5,
		0xcc, 0x46, 0x49, 0xbc, 0x31, 0x1d, 0x20, 0xc6, 0xc7, 0x20, 0x0f, 0x78,
		0xda, 0x7d, 0x55, 0xfe, 0x09, 0x99, 0xf5, 0x93, 0xb7, 0x4f, 0xe6, 0x9e,
		0x92, 0x48, 0x53, 0x50, 0x2b, 0x23, 0x14, 0xd6, 0x1b, 0x41, 0xb5, 0x76,
		0x53, 0xbe, 0x82, 0xb0, 0x65, 0x1f, 0xca, 0xd2, 0x00, 0x0d, 0x29, 0x6b,
		0x08, 0xc7, 0x3b, 0xa4, 0x4e, 0x8f, 0xb9, 0x56, 0x0d, 0x49, 0x47, 0x02,
		0xff, 0x60, 0x9a, 0x63, 0x58, 0xd2, 0x76, 0x96, 0xd8, 0xdd, 0x52, 0xb8,
		0x1d, 0xfd, 0xed, 0x50, 0x17, 0x02, 0xe7, 0x55, 0xd9, 0xac, 0xd0, 0x7f,
		0x1a, 0xea, 0xae, 0x43, 0xcf, 0xb8, 0x35, 0x24, 0x71, 0xef, 0x8b, 0xfd,
		0x3b, 0x00, 0x00, 0xff, 0xff, 0xf3, 0xf2, 0x31, 0x73, 0x0a, 0x08, 0x00,
		0x00,
	}))

	if err != nil {
		panic("Decompression failed: " + err.Error())
	}

	var b bytes.Buffer
	io.Copy(&b, gz)
	gz.Close()

	return b.Bytes()
}

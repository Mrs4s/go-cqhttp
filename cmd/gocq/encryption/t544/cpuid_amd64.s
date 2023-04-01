//go:build amd64
// +build amd64

// Copyright (c) 2015 Klaus Post, released under MIT License. See LICENSE file.

// func cpuid(op uint32) (eax, ebx, ecx, edx uint32)
TEXT ·cpuid(SB), 7, $0
    XORQ CX, CX
    MOVL op+0(FP), AX
    CPUID
    MOVL AX, eax+8(FP)
    MOVL BX, ebx+12(FP)
    MOVL CX, ecx+16(FP)
    MOVL DX, edx+20(FP)
    RET

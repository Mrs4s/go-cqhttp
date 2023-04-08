//go:build amd64
// +build amd64

#include "textflag.h"

DATA  LC0<>+0(SB)/4, $1634760805
DATA  LC0<>+4(SB)/4, $857760878
DATA  LC0<>+8(SB)/4, $2036477234
DATA  LC0<>+12(SB)/4, $1797285236
GLOBL LC0<>(SB), NOPTR, $16

TEXT ·sub_a(SB), NOSPLIT, $0-48
    MOVQ ·a+0(FP), DI
    MOVQ ·b+24(FP), CX
    MOVQ    CX, DX
    MOVBLZX 3(CX), CX
    XORB    CX,  (DI)
    MOVBLZX 2(DX), CX
    XORB    CX, 1(DI)
    MOVBLZX 1(DX), CX
    XORB    CX, 2(DI)
    MOVBLZX (DX),  CX
    XORB    CX, 3(DI)
    MOVBLZX 7(DX), CX
    XORB    CX, 4(DI)
    MOVBLZX 6(DX), CX
    XORB    CX, 5(DI)
    MOVBLZX 5(DX), CX
    XORB    CX, 6(DI)
    MOVBLZX 4(DX), CX
    XORB    CX, 7(DI)
    MOVBLZX 11(DX),CX
    XORB    CX, 8(DI)
    MOVBLZX 10(DX),CX
    XORB    CX, 9(DI)
    MOVBLZX 9(DX), CX
    XORB    CX,10(DI)
    MOVBLZX 8(DX), CX
    XORB    CX,11(DI)
    MOVBLZX 15(DX),CX
    XORB    CX,12(DI)
    MOVBLZX 14(DX),CX
    XORB    CX,13(DI)
    MOVBLZX 13(DX),CX
    XORB    CX,14(DI)
    MOVBLZX 12(DX),DX
    XORB    DL,15(DI)
    RET

TEXT ·sub_b(SB), NOSPLIT, $0-48
    MOVQ ·a+0(FP), DI
    MOVQ ·b+24(FP), CX
    MOVQ    CX, DX
    MOVBLZX 3(CX), CX
    XORB    CX, (DI)
    MOVBLZX 6(DX), CX
    XORB    CX, 1(DI)
    MOVBLZX 9(DX), CX
    XORB    CX, 2(DI)
    MOVBLZX 12(DX),CX
    XORB    CX, 3(DI)
    MOVBLZX 7(DX), CX
    XORB    CX, 4(DI)
    MOVBLZX 10(DX),CX
    XORB    CX, 5(DI)
    MOVBLZX 13(DX),CX
    XORB    CX, 6(DI)
    MOVBLZX (DX), CX
    XORB    CX,7(DI)
    MOVBLZX 11(DX),CX
    XORB    CX,8(DI)
    MOVBLZX 14(DX),CX
    XORB    CX,9(DI)
    MOVBLZX 1(DX), CX
    XORB    CX,10(DI)
    MOVBLZX 4(DX), CX
    XORB    CX,11(DI)
    MOVBLZX 15(DX),CX
    XORB    CX,12(DI)
    MOVBLZX 2(DX), CX
    XORB    CX,13(DI)
    MOVBLZX 5(DX), CX
    XORB    CX,14(DI)
    MOVBLZX 8(DX), DX
    XORB    DL,15(DI)
    RET


TEXT ·sub_c(SB), NOSPLIT, $0-32
    MOVQ ·a+0(FP), DI
    MOVQ ·b+8(FP), SI
    MOVQ SI, AX
    MOVBLZX (SI), SI
    MOVL SI, CX
    ANDL $15, SI
    SHRL $4, CX
    SHLL $4, CX
    ADDL SI, CX
    MOVBLZX 1(AX), SI
    MOVLQSX CX, CX
    MOVBLZX (DI)(CX*1), CX
    MOVB CX, (AX)
    MOVL SI, CX
    ANDL $15, SI
    SHRL $4, CX
    SHLL $4, CX
    ADDL SI, CX
    MOVBLZX 2(AX), SI
    MOVLQSX CX, CX
    MOVBLZX (DI)(CX*1), CX
    MOVB CX, 1(AX)
    MOVL SI, CX
    ANDL $15, SI
    SHRL $4, CX
    SHLL $4, CX
    ADDL SI, CX
    MOVBLZX 3(AX), SI
    MOVLQSX CX, CX
    MOVBLZX (DI)(CX*1), CX
    MOVB CX, 2(AX)
    MOVL SI, CX
    ANDL $15, SI
    SHRL $4, CX
    SHLL $4, CX
    ADDL SI, CX
    MOVBLZX 4(AX), SI
    MOVLQSX CX, CX
    MOVBLZX (DI)(CX*1), CX
    MOVB CX, 3(AX)
    MOVL SI, CX
    ANDL $15, SI
    SHRL $4, CX
    SHLL $4, CX
    ADDL SI, CX
    MOVBLZX 5(AX), SI
    MOVLQSX CX, CX
    MOVBLZX (DI)(CX*1), CX
    MOVB CX, 4(AX)
    MOVL SI, CX
    ANDL $15, SI
    SHRL $4, CX
    SHLL $4, CX
    ADDL SI, CX
    MOVBLZX 6(AX), SI
    MOVLQSX CX, CX
    MOVBLZX (DI)(CX*1), CX
    MOVB CX, 5(AX)
    MOVL SI, CX
    ANDL $15, SI
    SHRL $4, CX
    SHLL $4, CX
    ADDL SI, CX
    MOVBLZX 7(AX), SI
    MOVLQSX CX, CX
    MOVBLZX (DI)(CX*1), CX
    MOVB CX, 6(AX)
    MOVL SI, CX
    ANDL $15, SI
    SHRL $4, CX
    SHLL $4, CX
    ADDL SI, CX
    MOVBLZX 8(AX), SI
    MOVLQSX CX, CX
    MOVBLZX (DI)(CX*1), CX
    MOVB CX, 7(AX)
    MOVL SI, CX
    ANDL $15, SI
    SHRL $4, CX
    SHLL $4, CX
    ADDL SI, CX
    MOVBLZX 9(AX), SI
    MOVLQSX CX, CX
    MOVBLZX (DI)(CX*1), CX
    MOVB CX, 8(AX)
    MOVL SI, CX
    ANDL $15, SI
    SHRL $4, CX
    SHLL $4, CX
    ADDL SI, CX
    MOVBLZX 10(AX), SI
    MOVLQSX CX, CX
    MOVBLZX (DI)(CX*1), CX
    MOVB CX, 9(AX)
    MOVL SI, CX
    ANDL $15, SI
    SHRL $4, CX
    SHLL $4, CX
    ADDL SI, CX
    MOVBLZX 11(AX), SI
    MOVLQSX CX, CX
    MOVBLZX (DI)(CX*1), CX
    MOVB CX, 10(AX)
    MOVL SI, CX
    ANDL $15, SI
    SHRL $4, CX
    SHLL $4, CX
    ADDL SI, CX
    MOVBLZX 12(AX), SI
    MOVLQSX CX, CX
    MOVBLZX (DI)(CX*1), CX
    MOVB CX, 11(AX)
    MOVL SI, CX
    ANDL $15, SI
    SHRL $4, CX
    SHLL $4, CX
    ADDL SI, CX
    MOVBLZX 13(AX), SI
    MOVLQSX CX, CX
    MOVBLZX (DI)(CX*1), CX
    MOVB CX, 12(AX)
    MOVL SI, CX
    ANDL $15, SI
    SHRL $4, CX
    SHLL $4, CX
    ADDL SI, CX
    MOVBLZX 14(AX), SI
    MOVLQSX CX, CX
    MOVBLZX (DI)(CX*1), CX
    MOVB CX, 13(AX)
    MOVL SI, CX
    ANDL $15, SI
    SHRL $4, CX
    SHLL $4, CX
    ADDL SI, CX
    MOVBLZX 15(AX), SI
    MOVLQSX CX, CX
    MOVBLZX (DI)(CX*1), CX
    MOVB CX, 14(AX)
    MOVL SI, CX
    ANDL $15, SI
    SHRL $4, CX
    SHLL $4, CX
    ADDL SI, CX
    MOVLQSX CX, CX
    MOVBLZX (DI)(CX*1), CX
    MOVB CX, 15(AX)
    RET 

TEXT ·sub_d(SB), NOSPLIT, $16-32
    MOVQ ·t+0(FP), BX
    MOVQ ·s+8(FP), SI
    MOVOU   (SI), X0
    MOVOU   X0, in-16(SP)
    MOVQ    SI, DI
    ADDQ    $15, DI
    MOVB    $16, CX
lop:
    LEAQ    -1(CX), AX
    XLAT
    MOVBLZX in-16(SP)(AX*1), AX
    STD
    STOSB
    LOOP    lop
    RET

TEXT ·sub_e(SB), NOSPLIT, $0-32
    MOVQ ·a+0(FP), DI
    MOVQ ·n+8(FP), SI
    MOVQ $4, AX
lop:
    MOVBQZX -4(SI)(AX*4), DX
    MOVBQZX -3(SI)(AX*4), CX
    MOVBQZX -2(SI)(AX*4), R10
    MOVBQZX -1(SI)(AX*4), R8
    LEAQ (DX)(DX*2), R9
    LEAQ (R9*2), R9
    LEAQ (CX)(CX*2), R11
    LEAQ (R11*2), R11
    LEAQ (R10)(R10*2), BX
    LEAQ (BX*2), BX
    MOVB DX, R13
    XORB CX, DX
    XORB R10, CX
    MOVB (DI)(R9*1), R12
    XORB 1(DI)(R11*1), R12
    XORB R8, R10
    XORB R12, R10
    MOVB R10, -4(SI)(AX*4)
    MOVB (DI)(R11*1), R10
    XORB 1(DI)(BX*1), R10
    XORB R8, R13
    XORB R10, R13
    MOVB R13, -3(SI)(AX*4)
    MOVB (DI)(BX*1), R10
    LEAQ (R8)(R8*2), R8
    LEAQ (R8*2), R8
    XORB 1(DI)(R8*1), R10
    XORB R10, DX
    MOVB DX, -2(SI)(AX*4)
    MOVB 1(DI)(R9*1), DX
    XORB (DI)(R8*1), DX
    XORB DX, CX
    MOVB CX, -1(SI)(AX*4)
    DECB AX
    JNZ lop
    RET

TEXT sub_ab(SB), NOSPLIT, $0-24
    MOVQ ·s+0(FP), DI
    MOVQ ·w+8(FP), SI
    MOVL SI, AX
    MOVL SI, CX
    MOVL SI, DX
    SHRL $28, AX
    SHRL $24, CX
    ANDL $15, CX
    SALL $4, AX
    ADDL CX, AX
    MOVBLZX SI, CX
    MOVBLZX (DI)(AX*1), AX
    MOVBLZX (DI)(CX*1), CX
    SALL $24, AX
    ORL CX, AX
    MOVL SI, CX
    SHRL $8, SI
    SHRL $8, CX
    ANDL $15, SI
    ANDL $240, CX
    ADDL SI, CX
    MOVBLZX (DI)(CX*1), CX
    SALL $8, CX
    ORL CX, AX
    MOVL DX, CX
    SHRL $16, DX
    SHRL $16, CX
    ANDL $15, DX
    ANDL $240, CX
    ADDL CX, DX
    MOVBLZX (DI)(DX*1), DX
    SALL $16, DX
    ORL DX, AX
    MOVQ AX, ·retval+16(FP)
    RET

TEXT ·sub_f(SB), NOSPLIT, $24-68
    MOVQ ·k+0(FP), DI
    MOVQ ·r+8(FP), SI
    MOVQ ·s+16(FP), DX
    MOVQ $·w+24(FP), CX
    MOVQ CX, R10
    MOVQ SI, R9
    MOVQ DX, R8
    MOVL $4, BX
    MOVL (DI), AX
    BSWAPL AX
    MOVL AX, (CX)
    MOVL 4(DI), AX
    BSWAPL AX
    MOVL AX, 4(CX)
    MOVL 8(DI), AX
    BSWAPL AX
    MOVL AX, 8(CX)
    MOVL 12(DI), AX
    BSWAPL AX
    MOVL AX, 12(CX)
    JMP inner
for:
    XORL -16(R10)(BX*4), AX
    MOVL AX, (R10)(BX*4)
    ADDQ $1, BX
    CMPQ BX, $44
    JE end
inner:
    MOVL -4(R10)(BX*4), AX
    TESTB $3, BX
    JNE for
    ROLL $8, AX
    MOVQ R8, 0(SP)
    MOVL AX, 8(SP)
    CALL sub_ab(SB)
    MOVQ 16(SP), AX
    LEAL -1(BX), DX
    SARL $2, DX
    MOVLQSX DX, DX
    XORL (R9)(DX*4), AX
    JMP for
end:
    RET

TEXT ·sub_aa(SB), NOSPLIT, $0-56
    MOVQ ·i+0(FP), DI
    MOVQ ·t+8(FP), SI
    MOVQ ·b+16(FP), DX
    MOVQ ·m+24(FP), CX
    MOVL DI, AX
    MOVLQSX DI, DI
    MOVQ SI, R8
    MOVQ DX, SI
    MOVBLZX (CX)(DI*1), CX
    ANDL $15, AX
    MOVBLZX (SI)(AX*1), SI
    MOVQ AX, DX
    MOVL CX, AX
    SALQ $9, DX
    ANDL $15, CX
    SHRB $4, AX
    MOVL SI, DI
    ADDQ R8, DX
    SALQ $4, CX
    ANDL $15, AX
    SHRB $4, DI
    ANDL $15, SI
    SALQ $4, AX
    ANDL $15, DI
    ADDQ DX, AX
    ADDQ CX, DX
    MOVBLZX (AX)(DI*1), AX
    SALL $4, AX
    ORB 256(SI)(DX*1), AX
    MOVQ AX, ·retval+48(FP)
    RET

// func transformInner(x *[0x15]byte, tab *[32][16]byte)
TEXT ·transformInner(SB), NOSPLIT, $0-16
    MOVQ ·x+0(FP), DI
    MOVQ ·tab+8(FP), SI
    MOVQ    DI, AX
    MOVL    $1, CX
    MOVQ    SI, DI
    MOVQ    AX, SI
lop:
    MOVBLZX (SI), R8
    LEAL    -1(CX), AX
    ADDQ    $1, SI
    ANDL    $31, AX
    MOVL    R8, DX
    SALL    $4, AX
    ANDL    $15, R8
    SHRB    $4, DX
    MOVBLZX DX, DX
    ADDL    DX, AX
    CDQE
    MOVBLSX (DI)(AX*1), AX
    SALL    $4, AX
    MOVL    AX, DX
    MOVL    CX, AX
    ADDL    $2, CX
    ANDL    $31, AX
    SALL    $4, AX
    ADDL    R8, AX
    CDQE
    ORB     (DI)(AX*1), DX
    MOVB    DX, -1(SI)
    CMPL    CX, $43
    JNE     lop
    RET

TEXT ·initState(SB), NOSPLIT, $0-64
    MOVQ ·c+0(FP), DI
    MOVQ ·key+8(FP), SI
    MOVQ ·data+32(FP), R8
    MOVQ ·counter+56(FP), AX
    MOVOA   LC0<>(SB), X0
    MOVUPS  X0, (DI)
    MOVOU   (SI), X1
    MOVOU   (DI), X3
    MOVUPS  X1, 16(DI)
    MOVOU   16(SI), X2
    MOVQ    AX, 48(DI)
    MOVUPS  X2, 32(DI)
    MOVQ    (R8), AX
    MOVUPS  X3, 64(DI)
    MOVQ    AX, 56(DI)
    MOVQ    48(DI), AX
    MOVUPS  X1, 80(DI)
    MOVUPS  X2, 96(DI)
    MOVUPS  X6,112(DI)
    RET

TEXT ·sub_ad(SB), NOSPLIT, $8-24
    MOVQ ·a+0(FP), DI
    MOVQ DI, AX
    MOVL 40(DI), R10
    MOVL 12(DI), R12
    MOVL 44(DI), BP
    MOVL 16(DI), DX
    MOVL (DI), R15
    MOVL 48(DI), R9
    MOVL 20(DI), SI
    MOVL 32(DI), R11
    ADDL DX, R15
    MOVL 4(DI), R14
    MOVL 52(DI), R8
    XORL R15, R9
    MOVL 24(DI), CX
    MOVL 8(DI), R13
    ROLL $16, R9
    ADDL SI, R14
    MOVL 36(DI), BX
    MOVL 56(DI), DI
    ADDL R9, R11
    XORL R14, R8
    ADDL CX, R13
    XORL R11, DX
    ROLL $16, R8
    XORL R13, DI
    ROLL $12, DX
    ADDL R8, BX
    ROLL $16, DI
    ADDL DX, R15
    XORL BX, SI
    ADDL DI, R10
    XORL R15, R9
    ROLL $12, SI
    XORL R10, CX
    ROLL $8, R9
    ADDL SI, R14
    ROLL $12, CX
    ADDL R9, R11
    XORL R14, R8
    ADDL CX, R13
    XORL R11, DX
    ROLL $8, R8
    XORL R13, DI
    ROLL $7, DX
    LEAL (BX)(R8*1), BX
    ROLL $8, DI
    MOVL DX, tmp0-8(SP)
    MOVL 28(AX), DX
    XORL BX, SI
    MOVL BX, tmp1-4(SP)
    MOVL R10, BX
    MOVL 60(AX), R10
    ROLL $7, SI
    ADDL DI, BX
    ADDL DX, R12
    ADDL SI, R15
    XORL R12, R10
    XORL BX, CX
    ROLL $16, R10
    ROLL $7, CX
    ADDL R10, BP
    ADDL CX, R14
    XORL BP, DX
    XORL R14, R9
    ROLL $12, DX
    ROLL $16, R9
    ADDL DX, R12
    XORL R12, R10
    ROLL $8, R10
    ADDL R10, BP
    XORL R15, R10
    ROLL $16, R10
    XORL BP, DX
    ADDL R9, BP
    ADDL R10, BX
    ROLL $7, DX
    XORL BP, CX
    XORL BX, SI
    ROLL $12, SI
    ADDL SI, R15
    XORL R15, R10
    MOVL R15, (AX)
    ROLL $8, R10
    ADDL R10, BX
    MOVL R10, 60(AX)
    XORL BX, SI
    MOVD BX, X1
    ROLL $7, SI
    ROLL $12, CX
    ADDL DX, R13
    XORL R13, R8
    ADDL CX, R14
    MOVL SI, 20(AX)
    ROLL $16, R8
    XORL R14, R9
    MOVL R14, 4(AX)
    ADDL R8, R11
    ROLL $8, R9
    XORL R11, DX
    ADDL R9, BP
    MOVL R9, 48(AX)
    ROLL $12, DX
    XORL BP, CX
    MOVD BP, X2
    ADDL DX, R13
    ROLL $7, CX
    PUNPCKLLQ X2, X1
    XORL R13, R8
    MOVL CX, 24(AX)
    ROLL $8, R8
    MOVL R13, 8(AX)
    ADDL R8, R11
    XORL R11, DX
    MOVD R11, X0
    ROLL $7, DX
    MOVL DX, 28(AX)
    MOVL R8, 52(AX)
    MOVL tmp0-8(SP), SI
    MOVL tmp1-4(SP), CX
    ADDL SI, R12
    XORL R12, DI
    ROLL $16, DI
    ADDL DI, CX
    XORL CX, SI
    MOVL SI, DX
    ROLL $12, DX
    ADDL DX, R12
    XORL R12, DI
    MOVL R12, 12(AX)
    ROLL $8, DI
    ADDL DI, CX
    MOVL DI, 56(AX)
    MOVD CX, X3
    XORL CX, DX
    PUNPCKLLQ X3, X0
    ROLL $7, DX
    PUNPCKLQDQ X1, X0
    MOVL DX, 16(AX)
    MOVUPS X0, 32(AX)
    RET

// func tencentCrc32(tab *crc32.Table, b []byte) uint32
TEXT ·tencentCrc32(SB), NOSPLIT, $0-40
    MOVQ ·tab+0(FP), DI
    MOVQ ·bptr+8(FP), SI
    MOVQ ·bngas+16(FP), DX
    TESTQ   DX, DX
    JE      quickend
    ADDQ    SI, DX
    MOVL    $-1, AX
lop:
    MOVBLZX (SI), CX
    ADDQ    $1, SI
    XORL    AX, CX
    SHRL    $8, AX
    MOVBLZX CX, CX
    XORL    (DI)(CX*4), AX
    CMPQ    SI, DX
    JNE     lop
    NOTL    AX
    MOVQ    AX, ·bngas+32(FP)
    RET
quickend:
    XORL    AX, AX
    RET


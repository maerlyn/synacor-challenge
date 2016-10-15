package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
)

type Stack struct {
	count int
	items []uint16
}

func (stack *Stack) Push(item uint16) {
	stack.items = append(stack.items[:stack.count], item)
	stack.count++
}

func (stack *Stack) Pop() uint16 {
	if stack.count == 0 {
		panic("empty stack")
	}

	stack.count--

	return stack.items[stack.count]
}

type VM struct {
	memory             [32768]uint16
	registers          [8]uint16
	stack              Stack
	instructionPointer uint16

	debug *log.Logger
	stdinReader *bufio.Reader
}

func (vm *VM) Init() {
	file, err := os.Create(os.Getenv("PWD") + "/debug.log")
	if err != nil {
		panic("Can't open debug.log")
	}

	vm.debug = log.New(file, "", log.LstdFlags)
	vm.stdinReader = bufio.NewReader(os.Stdin)
}

func (vm *VM) LoadBinary(filename string) {
	fmt.Printf("Loading %s\n", filename)

	file, err := os.Open(filename)
	if err != nil {
		panic("Can't open file: " + err.Error())
	}
	defer file.Close()

	stat, _ := file.Stat()

	if err = binary.Read(file, binary.LittleEndian, vm.memory[0:(stat.Size()/2)]); err != nil {
		panic("Read failed: " + err.Error())
	}

	vm.debug.Println("Loaded", stat.Size()/2, "bytes")
}

func (vm *VM) Run() {
	for {
		vm.debug.Println("-------------------------------------------------------")
		vm.debug.Println("Current address:", vm.instructionPointer)
		operation := vm.nextValue()
		vm.debug.Println("Read instruction:", operation)

		switch operation {
		case 0:
			vm.opHalt()
		case 1:
			vm.opSet()
		case 2:
			vm.opPush()
		case 3:
			vm.opPop()
		case 4:
			vm.opEq()
		case 5:
			vm.opGt()
		case 6:
			vm.opJmp()
		case 7:
			vm.opJt()
		case 8:
			vm.opJf()
		case 9:
			vm.opAdd()
		case 10:
			vm.opMult()
		case 11:
			vm.opMod()
		case 12:
			vm.opAnd()
		case 13:
			vm.opOr()
		case 14:
			vm.opNot()
		case 15:
			vm.opRmem()
		case 16:
			vm.opWmem()
		case 17:
			vm.opCall()
		case 18:
			vm.opRet()
		case 19:
			vm.opOut()
		case 20:
			vm.opIn()
		case 21:
			vm.opNoop()

		default:
			panic("Not implemented: " + strconv.Itoa(int(operation)))

		}
	}
}

func (vm *VM) isRegister(address uint16) bool {
	return 32768 <= address && address <= 32775
}

func (vm *VM) getNumberOrRegister(value uint16) uint16 {
	if vm.isRegister(value) {
		vm.debug.Printf("value %d points to register #%d holding %d\n", value, value-32768, vm.registers[value-32768])
		return vm.registers[value-32768]
	}

	return value
}

func (vm *VM) nextValue() uint16 {
	result := vm.memory[vm.instructionPointer]
	vm.instructionPointer++

	return result
}

func (vm *VM) nextRegister() uint16 {
	value := vm.nextValue()

	if !vm.isRegister(value) {
		panic(strconv.Itoa(int(value)) + " is not a register")
	}

	return value - 32768
}

func (vm *VM) debugOp(op string, paramcount uint16) {
	vm.debug.Println("OP: " + op)

	if paramcount != 0 {
		vm.debug.Printf("params: %+v\n", vm.memory[vm.instructionPointer:vm.instructionPointer+paramcount])
	}
}

//halt: 0
//  stop execution and terminate the program
func (vm *VM) opHalt() {
	vm.debugOp("halt", 0)
}

//set: 1 a b
//  set register <a> to the value of <b>
func (vm *VM) opSet() {
	vm.debugOp("set", 2)
	register := vm.nextRegister()
	value := vm.getNumberOrRegister(vm.nextValue())
	vm.debug.Printf("register #%d, value %d\n", register, value)

	value = vm.getNumberOrRegister(value)

	vm.registers[register] = value
}

//push: 2 a
//  push <a> onto the stack
func (vm *VM) opPush() {
	vm.debugOp("push", 1)
	value := vm.getNumberOrRegister(vm.nextValue())
	vm.debug.Printf("value %d\n", value)

	value = vm.getNumberOrRegister(value)

	vm.stack.Push(value)
}

//pop: 3 a
//  remove the top element from the stack and write it into <a>; empty stack = error
func (vm *VM) opPop() {
	vm.debugOp("pop", 1)
	register := vm.nextRegister()
	value := vm.stack.Pop()
	vm.debug.Printf("register #%d, value %d\n", register, value)

	vm.registers[register] = value
}

//eq: 4 a b c
//  set <a> to 1 if <b> is equal to <c>; set it to 0 otherwise
func (vm *VM) opEq() {
	vm.debugOp("eq", 3)
	register := vm.nextRegister()
	a := vm.getNumberOrRegister(vm.nextValue())
	b := vm.getNumberOrRegister(vm.nextValue())

	if a == b {
		vm.registers[register] = 1
	} else {
		vm.registers[register] = 0
	}
}

//gt: 5 a b c
//  set <a> to 1 if <b> is greater than <c>; set it to 0 otherwise
func (vm *VM) opGt() {
	vm.debugOp("gt", 3)
	register := vm.nextRegister()
	a := vm.getNumberOrRegister(vm.nextValue())
	b := vm.getNumberOrRegister(vm.nextValue())

	if a > b {
		vm.registers[register] = 1
	} else {
		vm.registers[register] = 0
	}
}

//jmp: 6 a
//  jump to <a>
func (vm *VM) opJmp() {
	vm.debugOp("jmp", 1)
	address := vm.getNumberOrRegister(vm.nextValue())
	vm.debug.Printf("old address is %d, new address is %d\n", vm.instructionPointer, address)

	vm.instructionPointer = address
}

//jt: 7 a b
//  if <a> is nonzero, jump to <b>
func (vm *VM) opJt() {
	vm.debugOp("jt, jump if nonzero", 2)
	value := vm.getNumberOrRegister(vm.nextValue())
	address := vm.getNumberOrRegister(vm.nextValue())
	vm.debug.Printf("value is %d, address is %d\n", value, address)

	value = vm.getNumberOrRegister(value)

	if value != 0 {
		vm.instructionPointer = address
	}
}

//jf: 8 a b
//  if <a> is zero, jump to <b>
func (vm *VM) opJf() {
	vm.debugOp("jf, jump if zero", 2)
	value := vm.getNumberOrRegister(vm.nextValue())
	address := vm.getNumberOrRegister(vm.nextValue()) % 32768
	vm.debug.Printf("value is %d, address is %d\n", value, address)

	value = vm.getNumberOrRegister(value)

	if value == 0 {
		vm.instructionPointer = address
	}
}

//add: 9 a b c
//  assign into <a> the sum of <b> and <c> (modulo 32768)
func (vm *VM) opAdd() {
	vm.debugOp("add", 3)
	register := vm.nextRegister()
	a := vm.getNumberOrRegister(vm.nextValue())
	b := vm.getNumberOrRegister(vm.nextValue())

	vm.registers[register] = (a + b) % 32768
}

//mult: 10 a b c
//  store into <a> the product of <b> and <c> (modulo 32768)
func (vm *VM) opMult() {
	vm.debugOp("mult", 3)
	register := vm.nextRegister()
	a := vm.getNumberOrRegister(vm.nextValue())
	b := vm.getNumberOrRegister(vm.nextValue())

	vm.registers[register] = (a * b) % 32768
}

//mod: 11 a b c
//  store into <a> the remainder of <b> divided by <c>
func (vm *VM) opMod() {
	vm.debugOp("mod", 3)
	register := vm.nextRegister()
	a := vm.getNumberOrRegister(vm.nextValue())
	b := vm.getNumberOrRegister(vm.nextValue())

	vm.registers[register] = (a % b) % 32768
}

//and: 12 a b c
//  stores into <a> the bitwise and of <b> and <c>
func (vm *VM) opAnd() {
	vm.debugOp("and", 3)
	register := vm.nextRegister()
	a := vm.getNumberOrRegister(vm.nextValue())
	b := vm.getNumberOrRegister(vm.nextValue())

	vm.registers[register] = (a & b) % 32768
}

//or: 13 a b c
//  stores into <a> the bitwise or of <b> and <c>
func (vm *VM) opOr() {
	vm.debugOp("or", 3)
	register := vm.nextRegister()
	a := vm.getNumberOrRegister(vm.nextValue())
	b := vm.getNumberOrRegister(vm.nextValue())

	vm.registers[register] = (a | b) % 32768
}

//not: 14 a b
//  stores 15-bit bitwise inverse of <b> in <a>
func (vm *VM) opNot() {
	vm.debugOp("not", 2)
	register := vm.nextRegister()
	b := vm.getNumberOrRegister(vm.nextValue())

	vm.registers[register] = (b ^ 0xffff) % 32768
}

//rmem: 15 a b
//  read memory at address <b> and write it to <a>
func (vm *VM) opRmem() {
	vm.debugOp("rmem", 2)
	register := vm.nextRegister()
	address := vm.getNumberOrRegister(vm.nextValue())

	vm.registers[register] = vm.memory[address]
}

//wmem: 16 a b
//  write the value from <b> into memory at address <a>
func (vm *VM) opWmem() {
	vm.debugOp("wmem", 2)
	address := vm.getNumberOrRegister(vm.nextValue())
	value := vm.getNumberOrRegister(vm.nextValue())

	vm.memory[address] = value
}

//call: 17 a
//  write the address of the next instruction to the stack and jump to <a>
func (vm *VM) opCall() {
	vm.debugOp("call", 1)
	address := vm.getNumberOrRegister(vm.nextValue())
	vm.stack.Push(vm.instructionPointer)

	vm.instructionPointer = address
}

//ret: 18
//  remove the top element from the stack and jump to it; empty stack = halt
func (vm *VM) opRet() {
	vm.debugOp("ret", 0)
	address := vm.stack.Pop()
	vm.debug.Println("popped address is", address)

	vm.instructionPointer = address
}

//out: 19 a
//  write the character represented by ascii code <a> to the terminal
func (vm *VM) opOut() {
	vm.debugOp("out", 1)
	char := vm.getNumberOrRegister(vm.nextValue())
	vm.debug.Printf("char is %c\n", char)

	fmt.Printf("%c", char)
}

//in: 20 a
//  read a character from the terminal and write its ascii code to <a>; it can be assumed that once input starts, it
//  will continue until a newline is encountered; this means that you can safely read whole lines from the keyboard and
//  trust that they will be fully read
func (vm *VM) opIn() {
	vm.debugOp("in", 1)
	register := vm.nextRegister()

	input, err := vm.stdinReader.ReadByte()

	if err != nil && err != io.EOF {
		panic(err)
	}

	vm.registers[register] = uint16(input)
}

//noop: 21
//  no operation
func (vm *VM) opNoop() {
	vm.debugOp("noop", 0)
}

func main() {
	if len(os.Args) < 2 {
		panic("Missing argument: binary name")
	}

	var vm VM

	vm.Init()
	vm.LoadBinary(os.Getenv("PWD") + "/" + os.Args[1])
	vm.Run()
}

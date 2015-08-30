# goalife

The general idea is that you instantiate some sort of data structure representing a digital world, populate it with randomly-generated computer programs that have the ability to interact with the world, including the ability to reproduce, and throw in random mutations.  Eventually one of those randomly-generated organisms will spontaneously develop the ability to consume energy in the local environment, move and reproduce.  Eventually that organism will discover the ability to sense for energy and optimize its movements until eventually the programs become eerily like actual organisms running around the display.

## Running

This requires Go 1.4 or later.

    # Set up empty workspace, if needed
    mkdir workspace; cd workspace
    export GOPATH=$PWD

    go get github.com/dnesting/alife/goalife
    GOMAXPROCS=10 bin/goalife

You may need to widen your terminal to at least 200x55 characters.

## Explanation

The world is a 2D grid.  Each cell can be occupied by an Organism (see entities/org and entities/org/cpuorg), or a Food pellet (entities/food), or may be empty.  The CpuOrganism type is a simple 8-bit virtual machine with four registers, with opcodes described in entities/org/cpuorg/ops.go.  Among its operations are ops that move the organism forward, turn it left or right, eat the entity directly ahead (organism or food), reproduce, etc.

Each organism's program is executed in a separate goroutine, which permits great use of CPU resources.  Vestiges of an isochronous approach remain in the code.

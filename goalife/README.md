# goalife

[![GoDoc](https://godoc.org/github.com/dnesting/alife/goalife?status.svg)](https://godoc.org/github.com/dnesting/alife/goalife)

This is a hobby implementation (and somewhat of a framework) for evolving "digital life", a form of artificial life.  The implementation here consists of:

- [grid2d](grid2d) - a 2D discrete grid world, within which organisms are placed and live out their lives
- [grid2d/org](grid2d/org) - an "organism", with a store of energy, and methods for sensing and manipulating the world
- [grid2d/org/cpu1](grid2d/org/cpu1) - a simple virtual machine, with opcodes that call methods of an organism so that the VM itself drives the organism

The organism is capable of reproduction (via a specific opcode in cpu1), and the child will have a small chance of a random mutation.  This permits evolution of the bytecode.

The goal of the organism is basic: to come up with a strategy to consume food (either corpses of organisms or by stealing energy from living organisms) efficiently.  It's anticipated that programs will evolve the ability to search for high concentrations of food, turn and move in the direction of food, and consume food when they find it.

## Running

This requires Go 1.4 or later.

    # Set up empty workspace, if needed
    mkdir workspace; cd workspace
    export GOPATH=$PWD

    go get github.com/dnesting/alife/goalife
    bin/goalife

You may need to widen your terminal to at least 200x55 characters.

(Non-official) Hyparview (https://ieeexplore.ieee.org/abstract/document/4272993) implementation.

Implemented by using https://github.com/nm-morais/go-babel

# How to run

Go the the repository folder
Change file exampleConfig with desired configurations.

Run directly from code:

    $ go run main.go

Alternatively, compile and then run:

    $ go build .

    $ chmod +x hyparview

    ./hyparview

In order to select a random available port from the system :

    ./hyparview -rport

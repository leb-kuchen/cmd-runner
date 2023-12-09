# cmd-runner

## About
Run programs concurrently, pipe input to stdout and exit programs on Ctrl-C interrupt.

I created it to have a simple solution to run multiple programs concurrently in watch mode 
and some programs require the stdout pipe. 

## Usage
Simply define the programs as command line arguments. There is no need to define jobs or threads.
All programs exit, if one does.

## Install
```bash
go install github.com/leb-kuchen/cmd-runner@latest
```

Or download the binaries in the release section.

## Example
```bash
cmd-runner 'air' 'tailwind -i css/main.css -o public/output.css -w'
```




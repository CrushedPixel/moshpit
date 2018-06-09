# moshpit

A command-line tool for precise datamoshing using I-Frame removal.  
Comes with built-in scene cut detection for optimal results.

![tutorial](https://i.imgur.com/XGrR2Kv.gif)
![original](https://i.imgur.com/GJ0ahuC.gif)
![moshed](https://i.imgur.com/kZVTvIf.gif)

# Installation
Aside from the *moshpit* binary, which can be downloaded from the [releases page](https://github.com/CrushedPixel/moshpit/releases),
you need a copy of [*FFmpeg*](https://www.ffmpeg.org/) installed on your machine.  

# Usage

## Arguments
```
moshpit [options] <file>
```
*moshpit* takes the video file you want to mosh as the last argument.

| Option  | Description                                                               |
|---------|---------------------------------------------------------------------------|
| -ffmpeg | Specifies the location of the FFmpeg binary. Default: **ffmpeg**          |
| -log    | Specifies the target location of the FFmpeg log file. Default: no logging |


## Commands
After starting moshpit, you can use the following commands to create a datamoshed video:

### scenes
```scenes <threshold>```

Datamoshing via I-Frame removal yields the best results when applied at scene cuts.
The `scenes` command finds scene cuts in the input file, using the *threshold* parameter
to determine the similarity of each frame with the preceding frame.

A *threshold* of `0.2` usually gives good results.

### mosh
```mosh <output> <frame> [frame...]```

Moshes the input file, writing it to the specified output file.  
I-Frame removal is performed at the given frame indices, 
with scene cuts previously detected using the `scenes` command being suggested.  
Using `all` as a frame parameter performs I-Frame removal at all previously detected scene cuts.

### exit
Exits moshpit.  
Moshpit can also be terminated at any time using `Ctrl+C` (`SIGINT`).

# Building from source
Thanks to golang's flawed dependency system, setting up *moshpit* locally for development
is a bit of a hassle.
You can't use `go get`, as we're using modified versions (forks) of some libraries.
Therefore, you need to clone the repository into your *GOPATH* and use [`go dep`](https://github.com/golang/dep)
to install dependencies into the `vendor` directory:

```
mkdir -p $GOPATH/src/github.com/crushedpixel
cd $GOPATH/src/github.com/crushedpixel
git clone https://github.com/CrushedPixel/moshpit
dep ensure
``` 

You should now be able to build the `moshpit` command binary:

```
go build github.com/crushedpixel/moshpit/cmd/moshpit
```

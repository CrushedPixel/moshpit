# moshpit

**A fork of the original moshpit tool.**

It now uses Go modules. I might be updating it with more features at some point.

---

A command-line tool for surgical I-Frame removal, so-called datamoshing.  
Comes with built-in scene cut detection for optimal results.

![tutorial](https://i.imgur.com/XGrR2Kv.gif)
![original](https://i.imgur.com/nWUzNkC.gif)
![moshed](https://i.imgur.com/qx776K6.gif)

## Table of contents
- [Installation](#installation)
- [Usage](#usage)
- [How it works](#how-it-works)
- [Building from source](#building-from-source)

## Installation
Aside from the *moshpit* binary, which can be downloaded from the [releases page](https://github.com/makeworld-the-better-one/moshpit/releases),
you need a copy of [*FFmpeg*](https://www.ffmpeg.org/) installed on your machine.  

## Usage
### Arguments
```
moshpit [options] <file>
```
*moshpit* takes the video file you want to mosh as the last argument.

| Option  | Description                                           | Default    |
|---------|-------------------------------------------------------|------------|
| -ffmpeg | Specifies the location of the FFmpeg binary.          | `ffmpeg`   |
| -log    | Specifies the target location of the FFmpeg log file. | no logging |


### Commands
After starting moshpit, you can use the following commands to create a datamoshed video:

#### scenes
```scenes <threshold>```

Datamoshing via I-Frame removal yields the best results when applied at scene cuts.
The `scenes` command finds scene cuts in the input file, using the *threshold* parameter
to determine the similarity of each frame with the preceding frame.

A *threshold* of `0.2` usually gives good results.

#### mosh
```mosh <output> <frame> [frame...]```

Moshes the input file, writing it to the specified output file.  
I-Frame removal is performed at the given frame indices, 
with scene cuts previously detected using the `scenes` command being suggested.  
Using `all` as a frame parameter performs I-Frame removal at all previously detected scene cuts.

#### exit
Exits moshpit.  
Moshpit can also be terminated at any time using `Ctrl+C` (`SIGINT`).

## How it works
### The theory behind datamoshing
[Source](http://datamoshing.com/2016/06/26/how-to-datamosh-videos/)

Modern compressed video files have very complex methods of reducing the amount of storage or bandwidth needed to display the video. To do this most formats, such as the *AVI format*, don't store the entire image for each frame.

Frames which store an entire picture are called I-frames (Intra-coded), and can be displayed without any additional information.

Frames which don’t contain the entire picture require information from other frames in order to be displayed, either previous or subsequent frames, these frames are called P-frames (Predicted) and B-frames (Bi-predictive). Instead of storing full pictures, these P-frames and B-frames contain data describing only the differences in the picture from the preceding frame, and/or from the next frame, this data is much smaller compared to storing the entire picture — especially in videos where there isn’t much movement.

When a video is encoded, or compressed, a combination of these types of frames are used. In most cases this means many P-frames with I-frames interspersed at regular intervals and where drastic visual changes in the video occur. More information on frame types can be found [here](https://en.wikipedia.org/wiki/Video_compression_picture_types).

If an I-frame is corrupted, removed or replaced, the data contained in the following P-frames is applied to the wrong picture. In the above video I-frames have been removed and so instead of scenes changing properly you see the motion from a new scene applied to a picture from a previous frame. This process of corrupting, removing or replacing I-frames is the video datamoshing technique that *moshpit* uses.

### What moshpit does
When running the `mosh` command, *moshpit* converts the input file into an *AVI file*,
placing *I-Frames* only at the frames specified by the user.
This is done because single frames can be very easily identified and changed in the AVI format.

Each of the *I-Frames* in the resulting AVI file is then replaced with the next *P-Frame*, which means that the moshed video has the same duration as the original video, as opposed to removing the *I-Frames*, which would cause the moshed video to be shorter.

Finally, the moshed AVI file is "baked", which means it's converted back into an *MP4 file*,
persisting the artifacts in the AVI file into a stable video file.

## Building from source

```shell
go get github.com/makeworld-the-better-one/moshpit/cmd/moshpit
```

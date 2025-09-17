# data2vid

data2vid is a proof-of-concept (PoC) Go module that provides a command-line interface (CLI) for encoding arbitrary data files (such as PDF, JPEG, ZIP, TXT, etc.) into MP4 video files and subsequently decoding them back to their original form.  

The tool works by converting raw binary data directly into mp4 video black & white frames.  

## Requirements

Ffmpeg standard CLI should be installed on the user OS => https://ffmpeg.org/  

Go 1.20+ (for building from source)  


## Usage

1- Building the binary  

```go
go build -o data2vid
```

2- Encoding a data file into an mp4  

```go
./data2vid encode files_test/6mb.pdf -o 6mb.mp4
```

3- Decoding the original data from the mp4  
```go
./data2vid decode 6mb.mp4 -o original.pdf
```

## Configuration  

🔒 Fixed Parameters:  
  - Framerate	1 FPS	
  - Codec	libx264	H.264 video encoding  
  - Preset	ultrafast	Encoding speed/quality tradeoff  
  - Pixel Format	yuv420p	Widely compatible color space  

📏 Adjustable (via `config.yaml`):  
  - Frame Width -> Default: 1280 Pixels   
  - Frame Height -> Default: 720 Pixels    
  

#

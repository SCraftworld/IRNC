# IRNC
GUI application for Raspberry PI which allows simultaneous manipulation of infrared and nightvision cameras.
IRNC stands for "InfraRed and Nightvision/Normal (cameras) Combined".

# Why
After smartphone vendors switched from USB Micro to USB type C my old IR camera became obsolete (since usage of micro <-> type C USB adapters with it is cumbersome and unsafe).
This project was conceived as an opportunity for it to become useful (by being usable) again. Additional functionality in the form of second camera preview further enhances UX in cases when monitored object is not easily spotted in near infrared spectrum.

# Functionality
Shows fullscreen window with previews from both cameras and a line of control buttons (optimized for hand movement in freezing conditions).
You can save photo or save short (1min) video. Images/video captured simultaneously from both cameras which provides capacity for later comparison.
Video stream fed through V4L2 which may require additional setup (not included in application). Application intented to work with certain hardware configuration which means that following parameters are hardcoded:
- Screen resolution
- Camera type and resolution
- Physical camera location

# Deps
- Fyne.io for GUI
- libseek-thermal for interaction with IR camera
- ffmpeg for video encoding/decoding
- v4l2loopback-dkms for loopback device

# Setup
1. Install deps
2. Enable picam from raspi-config
3. Export path to libseek-thermal executables
4. Configure screen
```
gpio -g pwm 18 1024
gpio -g mode 18 pwm
gpio pwmc 1000
```
5. Create v4l2 loopback device
```
sudo modprobe v4l2loopback video_nr=1
```
6. Configure autorun if needed

# Build
```
cd irnc
$GOPATH/bin/fyne bundle -package irnc -name rscPhotoPng resources\photo.png > bundle.go
$GOPATH/bin/fyne bundle -append -name rscVideoPng resources\video.png >> bundle.go
$GOPATH/bin/fyne bundle -append -name rscExitPng resources\exit.png >> bundle.go
cd ..
go build
```

# Hardware
- Raspberry Pi 3 Model B
- Waveshare 3.5 inch RPi LCD (B)
- RPI Camera H
- Seek Thermal Compact Pro for Android
- Unnamed USB adapter (USB type A m <-> USB Micro f)

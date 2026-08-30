# LPA3588 AHD device mapping

The following mapping was verified with one working AHD camera connected to
each physical input in turn:

| Physical input | Linux capture device |
| --- | --- |
| AHD1 | `/dev/video11` |
| AHD2 | `/dev/video12` |
| AHD3 | `/dev/video13` |
| AHD4 | `/dev/video14` |
| AHD5 | `/dev/video3` |
| AHD6 | `/dev/video2` |
| AHD7 | `/dev/video1` |
| AHD8 | `/dev/video0` |

This verifies each physical input independently. It does not yet verify eight
simultaneous cameras or the final multi-camera publishing configuration.

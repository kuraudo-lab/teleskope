#!/usr/bin/env python3
"""Encode paced GIF walkthroughs from reviewed, unaltered browser keyframes."""
import json
from pathlib import Path
import subprocess

ROOT = Path(__file__).resolve().parents[1]
ASSETS = ROOT / 'assets' / 'highlights'
SCENES = ('topology', 'evidence', 'fleet')
for scene in SCENES:
    for step in range(1, 4):
        frame = ASSETS / 'frames' / f'{scene}-{step:02d}.jpg'
        if not frame.is_file():
            raise SystemExit(f'Missing captured UI frame: {frame}')
    output = ASSETS / f'{scene}.gif'
    subprocess.run([
        'ffmpeg', '-hide_banner', '-loglevel', 'error', '-y',
        '-framerate', '1/3', '-start_number', '1',
        '-i', str(ASSETS / 'frames' / f'{scene}-%02d.jpg'),
        '-filter_complex',
        '[0:v]scale=1280:-1:flags=lanczos,split[a][b];'
        '[a]palettegen=stats_mode=full[p];[b][p]paletteuse=dither=bayer:bayer_scale=3',
        '-loop', '0', '-final_delay', '300', str(output)
    ], check=True)
    probe = json.loads(subprocess.check_output([
        'ffprobe', '-v', 'error', '-count_frames', '-show_entries',
        'stream=width,height,nb_read_frames:format=duration', '-of', 'json', str(output)
    ]))
    stream = probe['streams'][0]
    assert (stream['width'], stream['height']) == (1280, 720), probe
    assert int(stream['nb_read_frames']) == 3, probe
    assert abs(float(probe['format']['duration']) - 9) < .1, probe
    assert output.stat().st_size < 1_000_000, 'README GIF exceeds 1 MB budget'
    print(f'{output.relative_to(ROOT)}: 1280x720, 9 seconds, '
          f'{output.stat().st_size / 1024:.0f} KiB')

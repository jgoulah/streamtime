#!/bin/bash
# Start Xvfb, openbox, and x11vnc for Amazon scraper
# Safe to run multiple times - checks if processes are already running

DISPLAY_NUM=99

# Check if Xvfb is already running on this display
if ! pgrep -f "Xvfb :${DISPLAY_NUM}" > /dev/null; then
    echo "Starting Xvfb on display :${DISPLAY_NUM}..."
    Xvfb :${DISPLAY_NUM} -screen 0 1920x1080x24 &
    sleep 2
else
    echo "Xvfb already running on display :${DISPLAY_NUM}"
fi

# Check if openbox is already running on this display
if ! pgrep -f "openbox" > /dev/null; then
    echo "Starting openbox..."
    DISPLAY=:${DISPLAY_NUM} openbox &
    sleep 1
else
    echo "openbox already running"
fi

# Check if x11vnc is already running
if ! pgrep -f "x11vnc" > /dev/null; then
    echo "Starting x11vnc on port 5900..."
    x11vnc -display :${DISPLAY_NUM} -nopw -forever &
else
    echo "x11vnc already running"
fi

echo "VNC display :${DISPLAY_NUM} ready. Connect via VNC to port 5900."

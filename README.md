# 🎨 GUI for TGPT 🧠💬
GTK-based graphical user interface for TGPT on Linux.

![demo](UI-preview.gif)

## ✨ Features
- 🌙 **Dark theme** 
- 💬 **Font Editing** (removes un-necessery characters from output and gives solid text with some editing)
-  💬 **Chat history** with session timestamps (works but sometime history file gets erased)
- 🖼️ **Image generation mode** using `tgpt -img`
- 🔄 **Lazy loading** for chat history
- 🚀 **Lightweight & fast** – built with GTK3 and Python

## Installation ⏬

🟦 Arch Linux / Manjaro:
```sh
sudo pacman -S python-gobject gtk3 gdk-pixbuf tgpt
```
🚀 Run the GUI
```sh
python3 chatbot.py
```

# 🧩 Gonsole
A modern terminal-based code editor built with Go.  
Fast, minimal, and made for developers who live in the terminal.  

---

## ✨ Features
- Syntax highlighting (powered by **Chroma**)
- Built-in terminal (PTY shell)
- File explorer sidebar
- Undo / Redo system
- Search (`Ctrl+F`)
- Extensions support

---

## ⚙️ Run
```sh
go run ./cmd/gonsole
```

Or install globally:
```sh
go install github.com/Mohammad-Alipour/Gonsole/cmd/gonsole@latest
```

Then run:
```sh
gonsole <file>
```

---

## ⌨️ Keybindings
| Action | Shortcut |
|--------|-----------|
| Save | `Ctrl + S` |
| Undo / Redo | `Ctrl + Z` / `Ctrl + Y` |
| Search | `Ctrl + F` |
| Toggle Terminal | `Ctrl + T` |
| Toggle Extensions | `Ctrl + E` |
| Switch Sidebar / Editor | `Tab` |
| Quit | `Ctrl + C` / `Esc` |

---

## 🗂️ Project Structure
```text
gonsole/
├── cmd/
│   └── gonsole/
│       └── main.go
├── internal/
│   ├── editor/
│   ├── syntax/
│   ├── lsp/
│   ├── ui/
│   └── config/
└── README.md
```

---

## 🖼️ Preview
*(Coming soon...)*

---

## 📜 License
MIT License © [Mohammad Alipour](https://github.com/Mohammad-Alipour)

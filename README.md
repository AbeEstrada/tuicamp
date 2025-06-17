![Icon](https://github.com/user-attachments/assets/27cad258-8627-4163-b4b6-73ac27accc4f)

# TuiCamp

Unofficial TimeCamp TUI

## Build and Installation

This project uses [`just`](https://github.com/casey/just) as a command runner and requires [Go](https://golang.org/) for building. Below are the available commands:

### Available Commands

- `just` or `just install` - Build and install the binary to `PREFIX/bin/` (default: `/usr/local/bin`)
- `just build` - Build the binary in the current directory
- `just uninstall` - Remove the installed binary
- `just clean` - Remove the built binary from the current directory

## Usage

```sh
tuicamp
```

## Keybindings

| Panel        |          Key           | Action                                       |
| :----------- | :--------------------: | :------------------------------------------- |
| All          |          `q`           | Quit                                         |
| Calendar     |       `h` or `←`       | Move to previous day                         |
| Calendar     |       `l` or `→`       | Move to next day                             |
| Calendar     |       `j` or `↓`       | Move to next week                            |
| Calendar     |       `k` or `↑`       | Move to previous week                        |
| Calendar     |          `L`           | Move to left panel (Timer)                   |
| Calendar     |          `J`           | Move to bottom panel (Entries)               |
| Calendar     |     `g` or `Home`      | Move to first day of month                   |
| Calendar     |      `G` or `End`      | Move to last day of month                    |
| Calendar     |    `p` or `Page Up`    | Move to previous month                       |
| Calendar     |   `n` or `Page Down`   | Move to next month                           |
| Calendar     |          `t`           | Move to today                                |
| Calendar     |   `Enter` or `Space`   | Select day                                   |
| Timer        |          `H`           | Move to right panel (Calendar)               |
| Timer        |          `J`           | Move to bottom panel (Entries)               |
| Timer        |   `Enter` or `Space`   | Start or stop timer                          |
| Entries      |          `K`           | Move to top panel (Calendar)                 |
| Entries      |       `j` or `↓`       | Move to next entry                           |
| Entries      |       `k` or `↑`       | Move to previous entry                       |
| Entries      |     `e` or `Enter`     | Edit entry                                   |
| Entries      |          `d`           | Delete entry                                 |
| Entry Edit   |      `q` or `Esc`      | Cancel editing and return                    |
| Entry Edit   |         `Tab`          | Cycle between time fields and task selection |
| Entry Edit   |   `Enter` or `Space`   | Save entry changes                           |
| Entry Edit   |      `Backspace`       | Delete last time digit                       |
| Entry Edit   |    `0-9` (numbers)     | Enter time digits (auto-formats as HH:MM:SS) |
| Entry Edit   |          `/`           | Search tasks                                 |
| Entry Edit   |       `j` or `↓`       | Move to next task                            |
| Entry Edit   |       `k` or `↑`       | Move to previous task                        |
| Entry Edit   |     `g` or `Home`      | Move to first task                           |
| Entry Edit   |      `G` or `End`      | Move to last task                            |
| Search tasks | `Esc`, `Enter`, or `/` | Exit search mode                             |
| Search tasks |      `Backspace`       | Delete last search character                 |
| Search tasks |     Any character      | Add to search query                          |

## Screenshots

![Calendar](https://github.com/user-attachments/assets/2ac68a9a-4ae2-4a7a-8dd4-fcc4db1e032a)

![Entries](https://github.com/user-attachments/assets/282c72c8-d3ed-44a5-b22e-89b29db186f4)

![Tasks](https://github.com/user-attachments/assets/068fabab-ff5e-45b7-bdb0-c92fcd754a47)

![Search](https://github.com/user-attachments/assets/d7fde81c-0b27-4e29-81fa-e4703f573c34)

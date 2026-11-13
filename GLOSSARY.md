# demoinfocs-golang

A Go library that parses CS:GO and CS2 demo files and exposes them as a live game state.

## Language

**State binder**:
The in-package module that binds the game state to the demo's entities. Its families (rules, players, weapons/grenades, inferno, bomb) are method groups on the binder, not separate adapters.

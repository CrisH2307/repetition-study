# Running this project in Claude Code

## First time, on your Mac
```bash
git clone https://github.com/CrisH2307/repetition-study && cd repetition-study
git commit --allow-empty -m "Initialize repository" && git push -u origin main
git switch -c qa && git push -u origin qa
git switch -c feature/setup
unzip ~/Downloads/repetition-study-claude-kit.zip -d .
git add -A && git commit -m "Add Claude Code project kit" 
cp .env.example .env 2>/dev/null || touch .env   # add your API keys here
claude "/unit U0"
```
Review the U0 contract, push, open the PR into `qa` and merge it. Every other unit depends on it.

## Every unit after that
```bash
git switch qa && git pull
scripts/unit.sh U4      # new terminal for each unit
```

## During the build
| Need | Do |
| --- | --- |
| Progress and cuts | `/status` |
| Context past 50% | `/handoff U4`, then a new session, then `/unit U4` |
| Change a task | Edit `rpi/U4/PROMPT.md`, not the chat |
| New rule for every session | Add one line to `CLAUDE.md` |
| New unit | Add `rpi/<name>/PROMPT.md` and a `.claude/boundaries.json` entry |
| Jev split | `python3 scripts/jev_split.py` |

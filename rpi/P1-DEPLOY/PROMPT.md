Unit P1-DEPLOY, wave 4. Branch `chore/deploy`. Model: Sonnet 5.5, effort medium. Labels: `Issue`, `phase-1`, `wave-4`.
Files: `deploy/`, `docs/DEPLOY.md`, `.github/workflows/`, `rpi/P1-DEPLOY/`.
Waits for Cris's host choice. Config for that host in a Canadian region (D17, D32): app from the P1-IMAGE Dockerfile, managed Postgres, secrets set by Cris on the host (never in git), daily backups, and a written restore test Cris runs once. No paid step without Cris's go. Plain steps in `docs/DEPLOY.md`.
Done when: Cris can deploy by following `docs/DEPLOY.md`, and the restore test is written. PR into qa with the labels above. Never merge. Report under 200 words.

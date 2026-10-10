with open("backend/cmd/e2e/main.go", "r") as f:
    lines = f.readlines()

out = []
sa_block = []
in_sa_block = False
for line in lines:
    if "11.5 Service account created:" in line:
        in_sa_block = True
    
    if in_sa_block:
        sa_block.append(line)
        if "12. Gateway credential created" in line:
            in_sa_block = False
            # sa_block contains the code that should be above gwResp
            # let's just rewrite the whole file cleanly.

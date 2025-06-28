# Experiments Tippecanoe

| Packages | Layer | Hours | Zoom| Duration |
|-----------|----------|----------|----------|
| SP1 | rainfall_accumulation| 1h | 5-9 | 3m09s |
| SP1 | rainfall_accumulation| 1h | 7 | 2m04s |
| SP1, SP2 | all | 1h | 7 | 3m30s | 
| SP1, SP2 | all | 1h | 5-9 | 5m50s |
| SP1, SP2 | all | 2h in parallel | 7 | >10min |

Based on this we can extrapolate the following results from the best case scenario: 

| Extrapolation | Duration | Details |
|---------------|----------|----------|
| Full run (51h) | 4h57m | Based on 5m50s per hour |
| With CPU improvement | 3h43m | 25% faster |
| 2 servers | 1h51m | Split workload |
| 3 servers | 1h14m | Split workload |
| 4 servers | 57m | Split workload |


Things to note:
- Zoom 7.3 is the minimum zoom for the layer to be properly displayed, so we don't need to generate tiles for zoom 7 and below.
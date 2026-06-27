import pandas as pd

df1 = pd.DataFrame({'a': [1, 2]})
df2 = pd.DataFrame({'a': [3, 4]})
# pd.DataFrame.append was REMOVED in pandas 2.0 -> contradicted
result = pd.DataFrame.append(df1, df2, ignore_index=True)

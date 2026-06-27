"""Shared fixture Python code strings used across tests."""

IMPORT_PANDAS = "import pandas as pd\n"

ATTRIBUTE_APPEND = """\
import pandas as pd
pd.DataFrame.append
"""

ATTRIBUTE_MERGE = """\
import pandas as pd
df1 = pd.DataFrame()
df2 = pd.DataFrame()
result = pd.DataFrame.merge(df1, df2)
"""

MISSING_IMPORT = """\
import receipts_nonexistent_pkg_xyz
"""

NUMPY_UFUNC = """\
import numpy as np
np.add.reduce([1, 2, 3])
"""

KWARG_REMOVED = """\
import pandas as pd
pd.read_csv('f.csv', error_bad_lines=True)
"""

KWARGS_CALLABLE = """\
import pandas as pd
pd.DataFrame.merge(pd.DataFrame(), pd.DataFrame(), how='inner')
"""

# Parsing & handling custom net-messages

## Finding interesting messages

You can use the build tag `debugdemoinfocs` to find interesting net-messages.

Example: `go run myprogram.go -tags debugdemoinfocs -ldflags '-X github.com/markus-wa/demoinfocs-golang.debugUnhandledMessages=YES' | grep "UnhandledMessage" | sort | uniq -c`

<details>
<summary>Sample output</summary>

```
      1 UnhandledMessage: id=10 name=svc_ClassInfo
      1 UnhandledMessage: id=14 name=svc_VoiceInit
   9651 UnhandledMessage: id=17 name=svc_Sounds
      1 UnhandledMessage: id=18 name=svc_SetView
    227 UnhandledMessage: id=21 name=svc_BSPDecal
  12705 UnhandledMessage: id=27 name=svc_TempEntities
    514 UnhandledMessage: id=28 name=svc_Prefetch
  85308 UnhandledMessage: id=4 name=net_Tick
      2 UnhandledMessage: id=5 name=net_StringCmd
      3 UnhandledMessage: id=7 name=net_SignonState
      1 UnhandledMessage: id=8 name=svc_ServerInfo
```
</details>

## Configuring a `NetMessageCreator`

NetMessageCreators are needed for creating instances of net-messages that aren't parsed by default.

You need to add them to the `ParserConfig.AdditionalNetMessageCreators` map where the key is the message-ID as seen in the debug output.

Example: Bullet Decal (`BSPDecal`) messages

```go
import (
	"github.com/gogo/protobuf/proto"

	dem "github.com/markus-wa/demoinfocs-golang"
	"github.com/markus-wa/demoinfocs-golang/msg"
)

cfg := dem.DefaultParserConfig
cfg.AdditionalNetMessageCreators = map[int]dem.NetMessageCreator{
	int(msg.SVC_Messages_svc_BSPDecal): func() proto.Message {
		return new(msg.CSVCMsg_BSPDecal)
	},
}

p := dem.NewParserWithConfig(f, cfg)
```

## Registering net-message handlers

To register a handler for net-messages `Parser.RegisterNetMessageHandler()` can be used.

When using `Parser.ParseToEnd()` net-messages and events are dispatched asynchronously. To get around this you can use `Parser.ParseNextFrame()` instead.

Example:

```go
p.RegisterNetMessageHandler(func(m *msg.CSVCMsg_BSPDecal) {
	fmt.Printf("bullet decal at x=%f y=%f z=%f\n", m.Pos.X, m.Pos.Y, m.Pos.Z)
})
```

<details>
<summary>Sample output</summary>

```
bullet decal at x=-2046.000000 y=401.000000 z=1879.000000
bullet decal at x=397.903992 y=-1208.000000 z=1855.219971
bullet decal at x=-410.000000 y=-753.000000 z=1612.000000
bullet decal at x=-428.000000 y=-725.304016 z=1733.550049
bullet decal at x=2627.000000 y=90.000000 z=1613.000000
bullet decal at x=323.911987 y=1462.910034 z=1694.000000
bullet decal at x=497.000000 y=1500.000000 z=1842.000000
bullet decal at x=1160.000000 y=1646.000000 z=1868.339966
bullet decal at x=229.000000 y=2064.000000 z=1827.709961
bullet decal at x=1712.000000 y=818.000000 z=1613.189941
bullet decal at x=2108.000000 y=485.000000 z=1619.000000
bullet decal at x=997.250000 y=1151.619995 z=1867.930054
bullet decal at x=997.000000 y=1353.709961 z=2007.000000
bullet decal at x=997.000000 y=1606.790039 z=2007.000000
bullet decal at x=980.000000 y=1498.000000 z=1857.469971
bullet decal at x=2267.879883 y=-1015.000000 z=1815.430054
bullet decal at x=-13.000000 y=-148.000000 z=1613.000000
bullet decal at x=1810.000000 y=-505.575989 z=1675.479980
bullet decal at x=-109.000000 y=562.000000 z=1629.000000
bullet decal at x=1030.000000 y=717.000000 z=1613.000000
bullet decal at x=481.000000 y=611.000000 z=1705.000000
bullet decal at x=-63.000000 y=-152.000000 z=1819.000000
bullet decal at x=121.233002 y=373.753998 z=1614.000000
bullet decal at x=1472.000000 y=-1911.689941 z=1705.000000
bullet decal at x=812.000000 y=-1551.000000 z=1612.390015
bullet decal at x=1338.000000 y=-579.000000 z=1700.000000
bullet decal at x=1199.000000 y=-579.000000 z=1746.000000
bullet decal at x=800.000000 y=354.000000 z=1738.000000
bullet decal at x=481.000000 y=532.000000 z=1796.000000
bullet decal at x=481.000000 y=529.000000 z=1766.000000
bullet decal at x=-335.000000 y=400.000000 z=1730.000000
bullet decal at x=1872.000000 y=809.000000 z=1613.000000
bullet decal at x=-450.000000 y=2009.000000 z=1796.660034
bullet decal at x=-624.000000 y=1458.000000 z=1941.750000
bullet decal at x=1337.000000 y=1338.000000 z=1861.000000
bullet decal at x=2861.000000 y=-238.000000 z=1613.000000
bullet decal at x=-927.000000 y=696.000000 z=1641.959961
bullet decal at x=2231.000000 y=-275.000000 z=1749.000000
bullet decal at x=2231.000000 y=-280.000000 z=1727.000000
bullet decal at x=833.000000 y=-325.000000 z=1749.000000
bullet decal at x=833.000000 y=-330.000000 z=1727.000000
bullet decal at x=1347.000000 y=-364.000000 z=1744.000000
bullet decal at x=1347.000000 y=-370.000000 z=1767.000000
bullet decal at x=2955.000000 y=336.000000 z=1614.000000
bullet decal at x=1393.000000 y=952.000000 z=1617.000000
bullet decal at x=247.000000 y=-1920.000000 z=1912.000000
bullet decal at x=1074.359985 y=1465.109985 z=1701.000000
bullet decal at x=1057.000000 y=2163.000000 z=1851.000000
bullet decal at x=847.000000 y=1822.000000 z=1857.430054
bullet decal at x=-825.000000 y=1462.000000 z=1974.000000
bullet decal at x=-692.000000 y=1462.000000 z=1974.000000
bullet decal at x=1277.000000 y=-435.000000 z=1898.000000
bullet decal at x=1023.729980 y=-435.000000 z=1897.760010
bullet decal at x=239.000000 y=-1232.219971 z=2078.739990
bullet decal at x=239.000000 y=-1254.420044 z=2061.300049
bullet decal at x=239.000000 y=-1355.150024 z=2040.979980
bullet decal at x=2859.120117 y=-611.000000 z=1874.000000
bullet decal at x=2604.000000 y=-786.000000 z=1874.000000
bullet decal at x=-1287.000000 y=1065.000000 z=1612.000000
bullet decal at x=-348.000000 y=613.000000 z=1763.170044
bullet decal at x=-1094.000000 y=-38.863800 z=1834.880005
bullet decal at x=-1094.000000 y=-248.740997 z=1834.520020
bullet decal at x=-1094.000000 y=-388.312988 z=1833.979980
bullet decal at x=-444.000000 y=-388.959991 z=1837.000000
bullet decal at x=-444.000000 y=-249.557007 z=1834.729980
bullet decal at x=-738.000000 y=423.574005 z=1834.560059
bullet decal at x=-1094.000000 y=425.303009 z=1834.430054
bullet decal at x=-127.000000 y=-1170.000000 z=1659.000000
bullet decal at x=25.000000 y=-1253.000000 z=1659.000000
bullet decal at x=629.000000 y=-907.000000 z=1615.000000
bullet decal at x=-265.000000 y=2110.000000 z=1687.000000
bullet decal at x=-463.812988 y=936.229980 z=1925.839966
bullet decal at x=2191.000000 y=556.249023 z=1619.000000
bullet decal at x=-45.000000 y=1882.000000 z=1687.000000
bullet decal at x=448.000000 y=-1377.000000 z=1740.000000
bullet decal at x=1322.000000 y=1688.000000 z=2537.540039
bullet decal at x=1753.000000 y=947.000000 z=1748.000000
bullet decal at x=1562.000000 y=-740.000000 z=1615.000000
bullet decal at x=1876.000000 y=1038.329956 z=2509.620117
bullet decal at x=-350.000000 y=499.730988 z=1920.000000
bullet decal at x=-350.000000 y=753.000000 z=1919.760010
bullet decal at x=2155.000000 y=566.000000 z=1613.000000
bullet decal at x=340.000000 y=1313.000000 z=1805.719971
bullet decal at x=-426.000000 y=-890.000000 z=1879.430054
bullet decal at x=-426.000000 y=-777.906006 z=1880.000000
bullet decal at x=1985.680054 y=-199.000000 z=1682.550049
bullet decal at x=-335.000000 y=-92.000000 z=1702.000000
bullet decal at x=-335.000000 y=-91.810303 z=1682.989990
bullet decal at x=229.000000 y=2046.000000 z=1835.199951
bullet decal at x=-100.168999 y=1098.709961 z=1688.000000
bullet decal at x=-75.413597 y=1099.000000 z=1688.000000
bullet decal at x=-396.000000 y=-403.000000 z=1773.000000
bullet decal at x=1232.000000 y=386.000000 z=1703.099976
bullet decal at x=-115.533997 y=-165.098007 z=1725.050049
bullet decal at x=508.058014 y=-591.114014 z=1612.000000
bullet decal at x=1440.000000 y=380.000000 z=1612.000000
bullet decal at x=-463.481995 y=200.882004 z=1662.000000
bullet decal at x=-452.000000 y=1503.000000 z=1823.000000
bullet decal at x=-609.403015 y=-149.000000 z=1813.819946
bullet decal at x=394.000000 y=1857.000000 z=1687.000000
bullet decal at x=880.000000 y=-681.370972 z=1712.020020
bullet decal at x=848.000000 y=1368.000000 z=1779.000000
bullet decal at x=865.296021 y=1456.000000 z=1702.000000
bullet decal at x=-792.866028 y=-1082.000000 z=1841.119995
bullet decal at x=-792.963013 y=-412.000000 z=1839.689941
bullet decal at x=580.000000 y=-658.000000 z=1748.510010
bullet decal at x=-537.000000 y=-785.000000 z=1884.000000
bullet decal at x=1131.000000 y=-1021.590027 z=1699.880005
bullet decal at x=731.000000 y=-856.000000 z=1760.949951
bullet decal at x=-51.684399 y=-1476.000000 z=1911.349976
bullet decal at x=880.000000 y=-766.000000 z=1711.060059
bullet decal at x=1557.000000 y=107.000000 z=1619.000000
bullet decal at x=1100.000000 y=351.000000 z=1688.609985
bullet decal at x=-415.367004 y=276.000000 z=1842.359985
bullet decal at x=-368.000000 y=-137.000000 z=1849.920044
bullet decal at x=-846.510010 y=-1084.000000 z=1723.750000
bullet decal at x=-738.000000 y=-94.622299 z=1925.290039
bullet decal at x=1246.000000 y=-152.000000 z=1772.780029
bullet decal at x=1522.000000 y=479.000000 z=1780.689941
bullet decal at x=997.000000 y=1288.000000 z=1770.000000
bullet decal at x=1322.000000 y=1632.310059 z=2130.639893
bullet decal at x=1322.000000 y=1659.280029 z=2129.479980
bullet decal at x=1322.000000 y=1761.400024 z=2129.489990
bullet decal at x=1322.000000 y=1789.020020 z=2129.840088
bullet decal at x=1322.000000 y=1631.030029 z=2242.040039
bullet decal at x=1322.000000 y=1659.060059 z=2242.070068
bullet decal at x=1322.000000 y=1760.709961 z=2241.409912
bullet decal at x=1322.000000 y=1788.829956 z=2241.510010
bullet decal at x=248.000000 y=-1919.000000 z=1958.900024
bullet decal at x=248.000000 y=-1920.000000 z=1977.900024
bullet decal at x=248.000000 y=-1920.000000 z=1993.569946
bullet decal at x=223.000000 y=-899.294006 z=1694.930054
bullet decal at x=223.000000 y=-885.825012 z=1722.550049
bullet decal at x=1194.000000 y=-1275.000000 z=1779.000000
bullet decal at x=-92.000000 y=-1475.000000 z=2003.000000
bullet decal at x=-428.000000 y=-1041.000000 z=1729.000000
bullet decal at x=-1212.430054 y=-291.000000 z=1898.229980
bullet decal at x=-1126.000000 y=7.659020 z=1696.900024
bullet decal at x=-335.000000 y=96.209396 z=1790.199951
bullet decal at x=-340.000000 y=814.000000 z=1716.000000
bullet decal at x=1159.000000 y=1630.000000 z=1869.719971
bullet decal at x=-404.677002 y=234.744995 z=1662.000000
bullet decal at x=-1111.000000 y=-271.000000 z=1987.000000
bullet decal at x=-43.000000 y=1199.000000 z=1772.540039
bullet decal at x=-42.000000 y=1198.150024 z=1803.180054
bullet decal at x=-920.000000 y=1460.000000 z=1828.000000
bullet decal at x=248.970993 y=495.000000 z=1807.119995
bullet decal at x=997.000000 y=985.307007 z=1747.869995
bullet decal at x=-650.000000 y=-761.822998 z=1694.849976
bullet decal at x=-650.000000 y=-762.801025 z=1726.599976
bullet decal at x=-1110.000000 y=90.000000 z=1932.650024
bullet decal at x=-1110.000000 y=309.000000 z=1929.790039
bullet decal at x=239.000000 y=-951.000000 z=1859.000000
bullet decal at x=71.000000 y=495.000000 z=1767.680054
bullet decal at x=938.000000 y=2278.000000 z=1848.000000
bullet decal at x=689.000000 y=2278.000000 z=1848.000000
bullet decal at x=440.091003 y=2278.000000 z=1847.770020
bullet decal at x=184.000000 y=2278.000000 z=1848.000000
bullet decal at x=938.000000 y=2278.000000 z=1782.099976
bullet decal at x=689.000000 y=2278.000000 z=1782.000000
bullet decal at x=439.946014 y=2278.000000 z=1782.000000
bullet decal at x=1030.000000 y=2028.000000 z=1802.150024
bullet decal at x=1030.729980 y=2027.000000 z=1887.000000
bullet decal at x=-552.000000 y=-103.000000 z=1662.000000
bullet decal at x=1508.339966 y=-543.659973 z=1892.000000
bullet decal at x=1508.339966 y=-546.340027 z=1911.000000
bullet decal at x=1508.339966 y=-546.340027 z=1927.000000
bullet decal at x=-1823.000000 y=603.000000 z=1730.000000
bullet decal at x=-1435.000000 y=522.000000 z=1613.209961
bullet decal at x=-1214.000000 y=272.000000 z=1614.000000
bullet decal at x=-1238.000000 y=167.000000 z=1614.000000
bullet decal at x=-63.000000 y=-152.000000 z=1887.000000
bullet decal at x=-63.000000 y=-152.000000 z=1906.000000
bullet decal at x=-63.000000 y=-152.000000 z=1922.000000
bullet decal at x=-625.000000 y=1458.000000 z=1987.000000
bullet decal at x=-625.000000 y=1458.000000 z=2006.000000
bullet decal at x=-625.000000 y=1458.000000 z=2022.000000
bullet decal at x=-1173.000000 y=867.000000 z=1926.000000
bullet decal at x=-1173.000000 y=867.000000 z=1945.000000
bullet decal at x=-1173.000000 y=867.000000 z=1961.000000
bullet decal at x=799.000000 y=354.000000 z=1806.719971
bullet decal at x=799.000000 y=354.000000 z=1825.719971
bullet decal at x=799.000000 y=354.000000 z=1841.719971
bullet decal at x=-250.000000 y=-602.000000 z=1612.000000
bullet decal at x=3026.000000 y=-262.000000 z=1630.000000
bullet decal at x=2510.000000 y=-55.000000 z=1613.000000
bullet decal at x=2366.000000 y=-735.000000 z=1613.000000
bullet decal at x=1743.000000 y=-988.000000 z=1613.000000
bullet decal at x=-0.000000 y=-232.000000 z=1785.000000
bullet decal at x=-1823.000000 y=-291.000000 z=1845.000000
bullet decal at x=878.075989 y=-1095.650024 z=1612.000000
bullet decal at x=-334.833008 y=-436.000000 z=1714.119995
bullet decal at x=-813.078003 y=907.145996 z=1673.770020
bullet decal at x=-211.000000 y=1548.000000 z=1686.000000
bullet decal at x=1553.000000 y=-152.000000 z=1741.589966
bullet decal at x=1572.500000 y=-975.744995 z=1613.000000
bullet decal at x=1270.410034 y=-244.205002 z=1613.000000
bullet decal at x=785.442017 y=1663.760010 z=1701.000000
bullet decal at x=-1046.250000 y=325.910004 z=1612.000000
bullet decal at x=-630.000000 y=-195.000000 z=1612.000000
bullet decal at x=904.000000 y=-104.000000 z=1612.000000
bullet decal at x=497.000000 y=1069.000000 z=1800.849976
bullet decal at x=1640.040039 y=992.000000 z=1855.969971
bullet decal at x=1962.000000 y=-1293.170044 z=1659.000000
bullet decal at x=2157.000000 y=-179.000000 z=1620.000000
bullet decal at x=2234.000000 y=-391.790009 z=1716.000000
bullet decal at x=2319.000000 y=190.565002 z=1716.810059
bullet decal at x=998.000000 y=845.487000 z=1670.510010
bullet decal at x=1869.000000 y=1362.000000 z=2392.000000
bullet decal at x=1860.000000 y=1216.609985 z=1953.369995
bullet decal at x=2044.069946 y=900.000000 z=1682.010010
bullet decal at x=2086.550049 y=900.000000 z=1679.619995
bullet decal at x=579.208984 y=1497.780029 z=1701.000000
bullet decal at x=783.000000 y=1118.000000 z=1870.000000
bullet decal at x=997.000000 y=1135.949951 z=1837.439941
bullet decal at x=997.000000 y=1121.609985 z=1867.930054
bullet decal at x=53.561699 y=-528.177002 z=1619.000000
bullet decal at x=-463.907990 y=938.000000 z=1905.000000
bullet decal at x=-64.000000 y=-1309.000000 z=1661.000000
bullet decal at x=1051.989990 y=145.643997 z=1612.000000
bullet decal at x=1598.550049 y=140.283997 z=1612.000000
bullet decal at x=-237.860001 y=1700.000000 z=1687.000000
bullet decal at x=-452.000000 y=1555.000000 z=1798.079956
bullet decal at x=-452.000000 y=1730.000000 z=1797.469971
bullet decal at x=-452.000000 y=1870.000000 z=1796.660034
bullet decal at x=-798.698975 y=122.702003 z=1612.000000
bullet decal at x=-1032.500000 y=4.463500 z=1612.000000
```
</details>

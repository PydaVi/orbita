// Boots a disposable local PDS + PLC, no Postgres, no TLS, no domain.
// Not the "stock" dev-env (bin.js boots Bluesky's entire test network,
// including the Bsky AppView + Ozone + Postgres) — here we only use the
// two pieces Beta 0 needs: TestPlc (identity resolution) and TestPds
// (repository).
import { TestNetworkNoAppView } from '@atproto/dev-env'

const network = await TestNetworkNoAppView.create({
  pds: { port: 2583 },
})

console.log('PLC (identity):', network.plc.url)
console.log('PDS (repository):', network.pds.url)
console.log('Admin auth header:', network.pds.adminAuthHeaders().authorization)
console.log('\nCtrl+C to tear everything down.')

process.on('SIGINT', async () => {
  console.log('\nshutting down...')
  await network.close()
  process.exit(0)
})

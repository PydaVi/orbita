// Sobe um PDS + PLC locais e descartáveis, sem Postgres, sem TLS, sem domínio.
// Não é o dev-env "de fábrica" (bin.js sobe a rede de teste inteira do Bluesky,
// incluindo Bsky AppView + Ozone + Postgres) — aqui usamos só as duas peças que
// o Beta 0 precisa: TestPlc (resolução de identidade) e TestPds (repositório).
import { TestNetworkNoAppView } from '@atproto/dev-env'

const network = await TestNetworkNoAppView.create({
  pds: { port: 2583 },
})

console.log('PLC (identidade):', network.plc.url)
console.log('PDS (repositório):', network.pds.url)
console.log('Admin auth header:', network.pds.adminAuthHeaders().authorization)
console.log('\nCtrl+C pra derrubar tudo.')

process.on('SIGINT', async () => {
  console.log('\nencerrando...')
  await network.close()
  process.exit(0)
})

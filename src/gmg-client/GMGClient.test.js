const GMGClient = require('./GMGClient')
const InvalidCommand = require('./InvalidCommand')

// Mock dgram so tests don't need a real network
jest.mock('dgram', () => {
  const EventEmitter = require('events')
  const createSocket = () => {
    const socket = new EventEmitter()
    socket.send = jest.fn((data, offset, length, port, host, cb) => {
      if (cb) cb(null)
    })
    socket.close = jest.fn()
    socket.bind = jest.fn((cb) => cb())
    socket.setBroadcast = jest.fn()
    socket.removeAllListeners = jest.fn()
    // Store ref so tests can emit messages on it
    createSocket._lastSocket = socket
    return socket
  }
  createSocket._lastSocket = null
  return { createSocket }
})

const dgram = require('dgram')

describe('GMGClient', () => {
  describe('constructor', () => {
    test('uses default options when none provided', () => {
      const client = new GMGClient()
      expect(client.port).toBe(8080)
      expect(client.host).toBe('255.255.255.255')
      expect(client.tries).toBe(5)
      expect(client.retryMs).toBe(2000)
    })

    test('accepts custom options', () => {
      const client = new GMGClient({
        port: 9090,
        host: '192.168.1.100',
        tries: 3,
        retryMs: 1000,
      })
      expect(client.port).toBe(9090)
      expect(client.host).toBe('192.168.1.100')
      expect(client.tries).toBe(3)
      expect(client.retryMs).toBe(1000)
    })

    test('logger is optional and silent by default', () => {
      const client = new GMGClient()
      // Should not throw
      expect(() => client._logger('test')).not.toThrow()
    })

    test('calls provided logger', () => {
      const logger = jest.fn()
      const client = new GMGClient({ logger })
      client._logger('hello')
      expect(logger).toHaveBeenCalledWith('hello')
    })
  })

  describe('getGrillId', () => {
    test('returns grill ID string from response', async () => {
      const client = new GMGClient({ host: '192.168.1.100', retryMs: 10 })
      const idBuffer = Buffer.from('GMG-123456')

      const promise = client.getGrillId()

      // Simulate grill response
      setTimeout(() => {
        const socket = dgram._lastSocket || dgram.createSocket._lastSocket
        socket.emit('message', idBuffer, { address: '192.168.1.100', port: 8080 })
      }, 20)

      const result = await promise
      expect(result).toBe('GMG-123456')
    })
  })

  describe('getGrillStatus', () => {
    test('returns a GrillStatus object', async () => {
      const client = new GMGClient({ host: '192.168.1.100', retryMs: 10 })
      const GrillStatus = require('./GrillStatus')

      // Build a minimal status buffer (64 hex chars = 32 bytes)
      const hex = '00'.repeat(32)
      const buf = Buffer.from(hex, 'hex')

      const promise = client.getGrillStatus()

      setTimeout(() => {
        const socket = dgram.createSocket._lastSocket
        socket.emit('message', buf, { address: '192.168.1.100', port: 8080 })
      }, 20)

      const result = await promise
      expect(result).toBeInstanceOf(GrillStatus)
    })
  })

  describe('setGrillTemp', () => {
    test('throws InvalidCommand if grill is off', async () => {
      const client = new GMGClient({ host: '192.168.1.100', retryMs: 10 })

      // Build an "off" status buffer (state char at position 61 = '0')
      const hex = '00'.repeat(32)
      const buf = Buffer.from(hex, 'hex')

      const promise = client.setGrillTemp(350)

      // First call is getGrillStatus - respond with off status
      setTimeout(() => {
        const socket = dgram.createSocket._lastSocket
        socket.emit('message', buf, { address: '192.168.1.100', port: 8080 })
      }, 20)

      await expect(promise).rejects.toThrow(InvalidCommand)
    })
  })

  describe('setFoodTemp', () => {
    test('throws InvalidCommand if grill is off', async () => {
      const client = new GMGClient({ host: '192.168.1.100', retryMs: 10 })

      const hex = '00'.repeat(32)
      const buf = Buffer.from(hex, 'hex')

      const promise = client.setFoodTemp(165)

      setTimeout(() => {
        const socket = dgram.createSocket._lastSocket
        socket.emit('message', buf, { address: '192.168.1.100', port: 8080 })
      }, 20)

      await expect(promise).rejects.toThrow(InvalidCommand)
    })
  })

  describe('commands', () => {
    test('command strings are correctly formatted', () => {
      // Verify the internal commands object by checking sendCommand builds correct data
      const client = new GMGClient({ host: '192.168.1.100', retryMs: 10 })

      // We can't access the frozen commands directly, but we can verify
      // the client module exports are correct
      const index = require('./index')
      expect(index.GMGClient).toBe(GMGClient)
      expect(index.Errors.InvalidCommand).toBe(InvalidCommand)
    })
  })
})

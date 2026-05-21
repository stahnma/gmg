const PollingManager = require('./PollingManager')

describe('PollingManager', () => {
  describe('constructor', () => {
    test('defaults', () => {
      const pm = new PollingManager()
      expect(pm.isPolling).toBe(false)
    })
  })

  describe('invokeWithRetry', () => {
    test('calls the task and returns result', async () => {
      const pm = new PollingManager({ tries: 1 })
      const task = jest.fn().mockResolvedValue('result')

      const result = await pm.invokeWithRetry(task)
      expect(result).toBe('result')
      expect(task).toHaveBeenCalled()
    })

    test('throws if task is not a function', async () => {
      const pm = new PollingManager()
      await expect(pm.invokeWithRetry(null)).rejects.toThrow('Task must be a function')
    })

    test('emits polled event on success', async () => {
      const pm = new PollingManager({ tries: 1 })
      const listener = jest.fn()
      pm.on('polled', listener)

      await pm.invokeWithRetry(() => 'data')
      expect(listener).toHaveBeenCalledWith('data')
    })
  })

  describe('start', () => {
    test('throws if task is not a function', async () => {
      const pm = new PollingManager()
      await expect(pm.start({ task: 'not a function' })).rejects.toThrow('Task must be a function')
    })

    test('throws if callback is not a function', async () => {
      const pm = new PollingManager()
      await expect(pm.start({ callback: 'nope' })).rejects.toThrow('Callback must be a function')
    })

    test('throws if runCondition is not a function', async () => {
      const pm = new PollingManager()
      await expect(pm.start({ runCondition: true })).rejects.toThrow('Run Condition must be a function')
    })

    test('throws if runCount is not a number', async () => {
      const pm = new PollingManager()
      await expect(pm.start({ runCount: 'five' })).rejects.toThrow('Run Count must be a number')
    })

    test('emits started event', async () => {
      const pm = new PollingManager({ pollingInterval: 10 })
      const listener = jest.fn()
      pm.on('started', listener)

      // Use runCount to stop after 1 iteration
      const task = jest.fn().mockResolvedValue('ok')
      await pm.start({ task, runCount: 1 })

      expect(listener).toHaveBeenCalled()
    })

    test('calls task and callback', async () => {
      const pm = new PollingManager({ pollingInterval: 10, tries: 1 })
      const task = jest.fn().mockResolvedValue('result')
      const callback = jest.fn()

      await pm.start({ task, callback, runCount: 1 })

      expect(task).toHaveBeenCalled()
      expect(callback).toHaveBeenCalledWith('result')
    })

    test('stops when runCondition returns false', async () => {
      const pm = new PollingManager({ pollingInterval: 10, tries: 1 })
      let callCount = 0
      const task = jest.fn().mockResolvedValue('ok')
      const runCondition = () => {
        callCount++
        return callCount < 2
      }

      await pm.start({ task, runCondition })
      expect(task).toHaveBeenCalledTimes(2)
    })
  })

  describe('stop', () => {
    test('emits stopped event', async () => {
      const pm = new PollingManager()
      const listener = jest.fn()
      pm.on('stopped', listener)

      await pm.stop()
      expect(listener).toHaveBeenCalled()
      expect(pm.isPolling).toBe(false)
    })
  })
})

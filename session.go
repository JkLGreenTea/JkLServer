package jklserver

import "sync"

type session struct {
	storage		map[interface{}]interface{}
	mutex		*sync.Mutex
	rwMutex 	*sync.RWMutex
}

func newSession() *session {
	return &session{
		storage:	 map[interface{}]interface{}{},
		mutex: 		 &sync.Mutex{},
		rwMutex: 	 &sync.RWMutex{},
	}
}

func (session *session) Insert(key interface{}, value interface{}) {
	session.mutex.Lock()
	session.storage[key] = value
	session.mutex.Unlock()
}

func (session *session) Delete(key interface{}) {
	session.mutex.Lock()
	delete(session.storage, key)
	session.mutex.Unlock()
}

func (session *session) Pop(key interface{}) interface{} {
	session.mutex.Lock()
	value := session.storage[key]
	delete(session.storage, key)
	session.mutex.Unlock()

	return value
}

func (session *session) Get(key interface{}) interface{} {
	session.mutex.Lock()
	value := session.storage[key]
	session.mutex.Unlock()

	return value
}

func (session *session) GetOk(key interface{}) (interface{}, bool) {
	session.mutex.Lock()
	value, ok := session.storage[key]
	session.mutex.Unlock()

	return value, ok
}
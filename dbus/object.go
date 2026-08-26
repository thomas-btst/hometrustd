package dbus

import (
	"fmt"

	"github.com/godbus/dbus/v5"
)

type Object struct {
	object dbus.BusObject
}

func newObject(object dbus.BusObject) *Object {
	return &Object{
		object: object,
	}
}

func (o *Object) Property(property Property) (any, error) {
	fullProperty := fmt.Sprintf("%s.%s", o.object.Destination(), property)
	variant, err := o.object.GetProperty(fullProperty)
	if err != nil {
		return nil, fmt.Errorf("failed to get property '%s' for path '%s': %w", property, o.object.Path(), err)
	}

	return variant.Value(), nil
}

func (o *Object) String(property Property) (string, error) {
	prop, err := o.Property(property)
	if err != nil {
		return "", err
	}

	str, ok := prop.(string)
	if !ok {
		return "", fmt.Errorf("failed to convert property '%s' on path '%s' to string: %v", property, o.object.Path(), prop)
	}

	return str, nil
}

func (o *Object) Bytes(property Property) ([]byte, error) {
	prop, err := o.Property(property)
	if err != nil {
		return nil, err
	}

	bytes, ok := prop.([]byte)
	if !ok {
		return nil, fmt.Errorf("failed to convert property '%s' on path '%s' to []byte: %v", property, o.object.Path(), prop)
	}

	return bytes, nil
}

func (o *Object) Uint32(property Property) (uint32, error) {
	prop, err := o.Property(property)
	if err != nil {
		return 0, err
	}

	u, ok := prop.(uint32)
	if !ok {
		return 0, fmt.Errorf("failed to convert property '%s' on path '%s' to uint32: %v", property, o.object.Path(), prop)
	}

	return u, nil
}

func (o *Object) Path(property Property) (Path, error) {
	prop, err := o.Property(property)
	if err != nil {
		return "", err
	}

	objectPath, ok := prop.(dbus.ObjectPath)
	if !ok {
		return "", fmt.Errorf("failed to convert property '%s' on path '%s' to ObjectPath: %v", property, o.object.Path(), prop)
	}

	return Path(objectPath), nil
}

func (o *Object) Paths(property Property) ([]Path, error) {
	prop, err := o.Property(property)
	if err != nil {
		return nil, err
	}

	objectPaths, ok := prop.([]dbus.ObjectPath)
	if !ok {
		return nil, fmt.Errorf("failed to convert property '%s' on path '%s' to []ObjectPath: %v", property, o.object.Path(), prop)
	}

	paths := make([]Path, len(objectPaths))
	for i, op := range objectPaths {
		paths[i] = Path(op)
	}

	return paths, nil
}

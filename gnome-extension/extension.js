import Gio from 'gi://Gio';
import GLib from 'gi://GLib';
import * as Main from 'resource:///org/gnome/shell/ui/main.js';
import {Extension} from 'resource:///org/gnome/shell/extensions/extension.js';

const BUS_NAME = 'org.ibus.bamboo.GnomeFocus';
const OBJECT_PATH = '/org/ibus/bamboo/GnomeFocus';
const INTERFACE_NAME = 'org.ibus.bamboo.GnomeFocus';

const INTERFACE_XML = `
<node>
  <interface name="org.ibus.bamboo.GnomeFocus">
    <method name="GetFocusedWindowClass">
      <arg name="wm_class" type="s" direction="out"/>
    </method>
    <method name="IsOverviewVisible">
      <arg name="visible" type="b" direction="out"/>
    </method>
  </interface>
</node>`;

export default class BambooFocusExtension extends Extension {
    constructor(metadata) {
        super(metadata);
        this._ownerId = 0;
        this._connection = null;
        this._registrationId = 0;
    }

    enable() {
        this._ownerId = Gio.bus_own_name(
            Gio.BusType.SESSION,
            BUS_NAME,
            Gio.BusNameOwnerFlags.NONE,
            connection => {
                this._connection = connection;
                const nodeInfo = Gio.DBusNodeInfo.new_for_xml(INTERFACE_XML);
                const interfaceInfo = nodeInfo.lookup_interface(INTERFACE_NAME);

                this._registrationId = connection.register_object(
                    OBJECT_PATH,
                    interfaceInfo,
                    (conn, sender, objectPath, interfaceName, methodName, parameters, invocation) => {
                        if (methodName === 'GetFocusedWindowClass') {
                            const window = global.display.focus_window;
                            const wmClass = window?.get_wm_class() ?? '';
                            invocation.return_value(new GLib.Variant('(s)', [wmClass]));
                        } else if (methodName === 'IsOverviewVisible') {
                            invocation.return_value(new GLib.Variant('(b)', [Main.overview.visible]));
                        } else {
                            invocation.return_dbus_error(
                                'org.freedesktop.DBus.Error.UnknownMethod',
                                `Unknown method: ${methodName}`
                            );
                        }
                    },
                    null,
                    null
                );
            },
            null,
            null
        );
    }

    disable() {
        if (this._connection && this._registrationId) {
            this._connection.unregister_object(this._registrationId);
        }
        this._registrationId = 0;
        this._connection = null;

        if (this._ownerId) {
            Gio.bus_unown_name(this._ownerId);
            this._ownerId = 0;
        }
    }
}

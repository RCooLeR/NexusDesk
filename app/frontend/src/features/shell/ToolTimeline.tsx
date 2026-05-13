import {Card} from '../../components/ui';
import type {ToolEvent} from '../../types';

type ToolTimelineProps = {
    events: ToolEvent[];
};

export function ToolTimeline({events}: ToolTimelineProps) {
    return (
        <Card className="timeline">
            <div className="pane-title">
                <span>Tool Timeline</span>
                <small>Visible by design</small>
            </div>
            {events.map((event) => (
                <div className="timeline-item" key={`${event.time}-${event.title}`}>
                    <time>{event.time}</time>
                    <strong>{event.title}</strong>
                    <p>{event.detail}</p>
                </div>
            ))}
        </Card>
    );
}
